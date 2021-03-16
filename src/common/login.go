package common

import (
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"

	uuid "github.com/satori/go.uuid"
)

// LoginLogicFunc 登陆公共逻辑
type LoginLogicFunc func(reply *pb.LoginReply) *pb.ErrorMessage

// Login 本机登录
func Login(account string, password string, mobile string, role pb.Roles, componentName string, extroInfo *pb.MessageExtroInfo, handleFunc LoginLogicFunc) (*pb.LoginReply, *pb.ErrorMessage) {
	if extroInfo.GetUserId() != "" {
		LogError("Login UserId not nil")
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	accountMutex, err := Locker.MessageLock(MessageLockAccountLogin+account, extroInfo, componentName)
	if err != nil {
		LogError("Login MessageLockAccount has err", err)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockAccountLogin+account, extroInfo, componentName, accountMutex)

	LogDebug("Login Login extroInfo", extroInfo)
	LogDebug("Login Login request", account, password)

	reply := &pb.LoginReply{}

	//返回全局配置
	reply.AllConfig = Configer.GetGlobalAll()

	reply.ServerTime = time.Now().Unix()
	mysqlRequest := &pb.MysqlAccountInfo{}
	mysqlRequest.Account = account
	mysqlRequest.Password = password
	mysqlRequest.Mobile = mobile
	mysqlReply := &pb.MysqlAccountInfo{}
	msgErr := Router.Call("Mysql", "VerifyAccount", mysqlRequest, mysqlReply, extroInfo)
	if msgErr != nil {
		LogError("Login Login Call Mysql VerifyAccount has err", account, password, msgErr)
		if msgErr.GetCode() == pb.ErrorCode_DataNotFound {
			return reply, GetGrpcErrorMessage(pb.ErrorCode_PasswordError, "")
		}
		return reply, msgErr
	}
	//判断用户角色
	if mysqlReply.RoleType != role ||
		mysqlReply.PlayerSourceType != pb.PlayerSourceType_PlayerSourceType_Local {
		return nil, GetGrpcErrorMessage(pb.ErrorCode_AccountNotFound, "")
	}

	LogDebug("Login Login VerifyAccount ok")

	uuid := mysqlReply.GetUuid()
	shortID := mysqlReply.GetShortId()

	//无论如何先尝试踢出之前在线的自己
	Pusher.Push(GetGrpcErrorMessage(pb.ErrorCode_LoginOnOther, ""), uuid)
	time.Sleep(200 * time.Millisecond)
	msgErr = Pusher.Kick(uuid)
	if msgErr != nil {
		LogError("Login Login KickSelfFailed", msgErr, uuid)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_KickSelfFailed, "")
	}

	playerMutex, err := Locker.MessageLock(MessageLockPlayer+uuid, extroInfo, componentName)
	if err != nil {
		LogError("Login Login  MessageLockAccount has err", err)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockPlayer+uuid, extroInfo, componentName, playerMutex)
	LogDebug("Login Login MessageLockPlayer ok")
	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr = Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	LogDebug("Login Login LoadPlayer ok")
	//clientConnIp := extroInfo.GetClientConnIp()
	//clientConnId := extroInfo.GetClientConnId()
	clientServerIndex := extroInfo.GetServerIndex()
	reply.PlayerInfo = loadPlayerReply.GetPlayerInfo()
	if msgErr != nil {
		if msgErr.GetCode() == pb.ErrorCode_DataNotFound {
			newPlayerRequest := &pb.NewPlayerRequest{}
			newPlayerRequest.Uuid = uuid
			newPlayerRequest.ShortId = shortID
			newPlayerRequest.RoleType = mysqlReply.GetRoleType()
			newPlayerRequest.Account = account
			newPlayerReply := &pb.NewPlayerReply{}
			msgErr = Router.Call("PlayerInfo", "NewPlayer", newPlayerRequest, newPlayerReply, extroInfo)
			if msgErr != nil {
				LogError("Login Login Call PlayerInfo NewPlayer has err", uuid, msgErr)
				return reply, msgErr
			}
			reply.PlayerInfo = newPlayerReply.GetPlayerInfo()
			/*msgErr := Pusher.SetOnline(uuid, clientServerIndex)
			if msgErr != nil {
				return reply, msgErr
			}*/
			/*msgErr = Pusher.UserLoginOk(uuid, mysqlReply.GetRoleType(), reply.PlayerInfo.GetAuths(), extroInfo.GetClientConnId(), clientServerIndex)
			if msgErr != nil {
				return reply, msgErr
			}
			LogDebug("Login Login NewPlayer ok")
			return reply, nil*/
		} else {
			LogError("Login Login Call PlayerInfo LoadPlayer has err", msgErr)
			return reply, msgErr
		}
	} else {
		reply.PlayerInfo.LastLoginTime = time.Now().Unix()
	}

	if reply.PlayerInfo.GetStatus() > pb.UserStatus_normal {
		return nil, GetGrpcErrorMessage(pb.ErrorCode_AccountDisabled, "")
	}
	LogDebug("Login Login check status ok")
	msgErr = handleFunc(reply)
	if msgErr != nil {
		return reply, msgErr
	}
	reply.PlayerInfo.LastLoginTime = time.Now().Unix()
	//登录相关全部完成-保存进redis
	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = reply.PlayerInfo
	savePlayerRequest.ForceSave = false
	msgErr = Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
	if msgErr != nil {
		LogError("Login login SavePlayer has err ", msgErr)
		return reply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	LogDebug("Login handleFunc ok")

	/*msgErr = Pusher.SetOnline(uuid, clientServerIndex)
	if msgErr != nil {
		return reply, msgErr
	}
	LogDebug("Login Setonline ok")*/
	msgErr = Pusher.UserLoginOk(uuid, mysqlReply.GetRoleType(), reply.PlayerInfo.GetAuths(), extroInfo.GetClientConnId(), clientServerIndex)
	if msgErr != nil {
		return reply, msgErr
	}

	if reply.PlayerInfo.GetName() == "" {
		reply.PlayerInfo.Name = reply.PlayerInfo.GetShortId()
	}

	//time.Sleep(1 * time.Second)
	/*go func() {
	err := common.Pusher.OnLine(uuid, mysqlReply.RoleType, reply.PlayerInfo.Auths, clientConnId, clientConnIp)
	if err != nil {
		common.LogError("Login Login OnLine has err", err)
		go common.Pusher.KickByConnId(clientConnId, clientConnIp)
	}

	/*time.Sleep(time.Duration(2) * time.Second)
	common.Pusher.Push(reply, "1")
	common.Pusher.Push(reply, "2")
	go common.Pusher.Broadcast(reply)
	time.Sleep(time.Duration(2) * time.Second)
	go common.Pusher.Kick("1")
	go common.Pusher.Kick("2")*/
	/*}()*/
	LogDebug("Login All ok")
	return reply, nil
}

// ThirdPartyLogin 第三方登录
func ThirdPartyLogin(
	accessInfo *pb.ThirdPartyAccessInfo,
	userInfo *pb.ThirdPartyUerInfo,
	sourceType pb.PlayerSourceType,
	componentName string, extroInfo *pb.MessageExtroInfo, handleFunc LoginLogicFunc) (*pb.LoginReply, *pb.ErrorMessage) {
	if extroInfo.GetUserId() != "" {
		LogError("Login UserId not nil")
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	var openId string
	var unionId string
	var nickName string
	var sex int32
	var headImgUrl string
	var country string
	var province string
	var city string
	var account string
	var token string
	var saveTokenInfo *TokenInfo
	switch sourceType {
	case pb.PlayerSourceType_PlayerSourceType_WeChat:
		unionId = accessInfo.WeChatAccessInfo.GetUnionId()
		openId = accessInfo.WeChatAccessInfo.GetOpenId()
		nickName = userInfo.WeChatUserInfo.GetNickName()
		sex = userInfo.WeChatUserInfo.GetSex()
		headImgUrl = userInfo.WeChatUserInfo.GetHeadImgUrl()
		country = userInfo.WeChatUserInfo.GetCountry()
		province = userInfo.WeChatUserInfo.GetProvince()
		city = userInfo.WeChatUserInfo.GetCity()
		account = uuid.NewV4().String()
		token = accessInfo.WeChatAccessInfo.AccessToken
		saveTokenInfo = &TokenInfo{
			AccessToken:  accessInfo.WeChatAccessInfo.AccessToken,
			TokenType:    "weChat",
			ExpiresIn:    accessInfo.WeChatAccessInfo.ExpiresIn,
			RefreshToken: accessInfo.WeChatAccessInfo.RefreshToken,
			Scope:        accessInfo.WeChatAccessInfo.Scope,
			WeChatExtendedData: &WeChatTokenExtendedData{
				OpenId:  accessInfo.WeChatAccessInfo.OpenId,
				UnionId: accessInfo.WeChatAccessInfo.UnionId,
			},
		}

	case pb.PlayerSourceType_PlayerSourceType_Alipay:
		//TODO 第三方登录增加逻辑
	case pb.PlayerSourceType_PlayerSourceType_QQ:
		//TODO 第三方登录增加逻辑
	case pb.PlayerSourceType_PlayerSourceType_Google:
		//TODO 第三方登录增加逻辑
	case pb.PlayerSourceType_PlayerSourceType_Facebook:
		unionId = ""
		openId = userInfo.FacebookUserInfo.OpenId
		nickName = userInfo.FacebookUserInfo.NickName
		if userInfo.FacebookUserInfo.Gender == "male" {
			sex = 1
		} else if userInfo.FacebookUserInfo.Gender == "female" {
			sex = 2
		} else {
			sex = 0
		}
		headImgUrl = userInfo.FacebookUserInfo.HeadImgUrl
		country = ""
		province = ""
		city = ""
		if userInfo.FacebookUserInfo.Email != "" {
			account = userInfo.FacebookUserInfo.Email
		} else {
			uuid.NewV4().String()
		}
		token = accessInfo.FacebookAccessInfo.AccessToken
	}

	accountMutex, err := Locker.MessageLock(MessageLockAccountLogin+"_"+strconv.Itoa(int(sourceType))+"_"+openId+"_"+unionId, extroInfo, componentName)
	if err != nil {
		LogError("Login MessageLockAccount has err", err)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockAccountLogin+"_"+strconv.Itoa(int(sourceType))+"_"+openId+"_"+unionId, extroInfo, componentName, accountMutex)

	LogDebug("Login Login extroInfo", extroInfo)

	reply := &pb.LoginReply{}

	//返回全局配置
	reply.AllConfig = Configer.GetGlobalAll()

	reply.ServerTime = time.Now().Unix()

	queryAccountRequest := &pb.QueryThirdPartyAccountRequest{}
	queryAccountRequest.PlayerSourceType = sourceType
	queryAccountRequest.OpenId = openId
	queryAccountRequest.UnionId = unionId

	queryAccountReply := &pb.QueryThirdPartyAccountReply{}

	msgErr := Router.Call("Mysql", "QueryThirdPartyAccount", queryAccountRequest, queryAccountReply, extroInfo)
	if msgErr != nil {
		return reply, msgErr
	}

	mysqlRequest := &pb.MysqlAccountInfo{}
	//如果玩家不存在，则新增
	if len(queryAccountReply.AccountInfos) == 0 {
		mysqlRequest.OpenId = openId
		mysqlRequest.UnionId = unionId
		mysqlRequest.Account = account
		mysqlRequest.Password = EncodePassword("123456")
		mysqlRequest.PlayerSourceType = sourceType
		mysqlRequest.RoleType = pb.Roles_Player

		getShortIDReq := &pb.GetShortIdRequest{}
		getShortIDReply := &pb.GetShortIdReply{}
		shortidErr := Router.Call("Shortid", "Get", getShortIDReq, getShortIDReply, extroInfo)
		if shortidErr != nil {
			//common.LogError("Register Register Call Mysql NewAccount has err", account, msgErr)
			return reply, shortidErr
		}
		mysqlRequest.ShortId = getShortIDReply.GetShortId()

		mysqlReply := &pb.MysqlAccountInfo{}
		msgErr := Router.Call("Mysql", "NewAccount", mysqlRequest, mysqlReply, extroInfo)
		if msgErr != nil {
			//common.LogError("Register Register Call Mysql NewAccount has err", account, msgErr)
			return reply, msgErr
		}
		mysqlRequest.Uuid = mysqlReply.GetUuid()
		mysqlRequest.ShortId = mysqlReply.GetShortId()
		LogDebug("Register Register ok", mysqlReply)
	} else {
		mysqlRequest = queryAccountReply.AccountInfos[0]
	}

	LogDebug("Login Login VerifyAccount ok")

	uuid := mysqlRequest.GetUuid()
	shortID := mysqlRequest.GetShortId()

	//无论如何先尝试踢出之前在线的自己
	Pusher.Push(GetGrpcErrorMessage(pb.ErrorCode_LoginOnOther, ""), uuid)
	time.Sleep(200 * time.Millisecond)
	msgErr = Pusher.Kick(uuid)
	if msgErr != nil {
		LogError("Login Login KickSelfFailed", msgErr, uuid)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_KickSelfFailed, "")
	}

	playerMutex, err := Locker.MessageLock(MessageLockPlayer+uuid, extroInfo, componentName)
	if err != nil {
		LogError("Login Login  MessageLockAccount has err", err)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockPlayer+uuid, extroInfo, componentName, playerMutex)
	LogDebug("Login Login MessageLockPlayer ok")
	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr = Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	LogDebug("Login Login LoadPlayer ok")
	clientServerIndex := extroInfo.GetServerIndex()
	reply.PlayerInfo = loadPlayerReply.GetPlayerInfo()
	if msgErr != nil {
		if msgErr.GetCode() == pb.ErrorCode_DataNotFound {
			newPlayerRequest := &pb.NewPlayerRequest{}
			newPlayerRequest.Uuid = uuid
			newPlayerRequest.ShortId = shortID
			newPlayerRequest.RoleType = mysqlRequest.GetRoleType()
			newPlayerRequest.Account = mysqlRequest.Account
			newPlayerRequest.Name = nickName
			newPlayerRequest.Sex = sex
			newPlayerRequest.HeadImgUrl = headImgUrl
			newPlayerRequest.Country = country
			newPlayerRequest.Province = province
			newPlayerRequest.City = city
			newPlayerRequest.ThirdPartyAccessInfo = accessInfo
			newPlayerReply := &pb.NewPlayerReply{}
			msgErr = Router.Call("PlayerInfo", "NewPlayer", newPlayerRequest, newPlayerReply, extroInfo)
			if msgErr != nil {
				return reply, msgErr
			}
			reply.PlayerInfo = newPlayerReply.GetPlayerInfo()

		} else {
			LogError("Login Login Call PlayerInfo LoadPlayer has err", msgErr)
			return reply, msgErr
		}
	} else {
		reply.PlayerInfo.Name = nickName
		reply.PlayerInfo.Sex = sex
		reply.PlayerInfo.HeadImgUrl = headImgUrl
		reply.PlayerInfo.Country = country
		reply.PlayerInfo.Province = province
		reply.PlayerInfo.City = city
		reply.PlayerInfo.ThirdPartyAccessInfo = accessInfo
		reply.PlayerInfo.LastLoginTime = time.Now().Unix()
	}

	if reply.PlayerInfo.GetStatus() > pb.UserStatus_normal {
		return nil, GetGrpcErrorMessage(pb.ErrorCode_AccountDisabled, "")
	}

	LogDebug("Login Login check status ok")
	msgErr = handleFunc(reply)
	if msgErr != nil {
		return reply, msgErr
	}
	reply.PlayerInfo.LastLoginTime = time.Now().Unix()
	//登录相关全部完成-保存进redis
	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = reply.PlayerInfo
	savePlayerRequest.ForceSave = false
	msgErr = Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
	if msgErr != nil {
		LogError("Login login SavePlayer has err ", msgErr)
		return reply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	LogDebug("Login handleFunc ok")

	msgErr = Pusher.UserLoginOk(uuid, mysqlRequest.GetRoleType(), reply.PlayerInfo.GetAuths(), extroInfo.GetClientConnId(), clientServerIndex)
	if msgErr != nil {
		return reply, msgErr
	}
	LogDebug("Login All ok")
	reply.PlayerInfo.ThirdPartyAccessInfo = nil
	reply.AccessToken = token
	if saveTokenInfo != nil {
		Tokener.SaveThirdPartyToken(reply.PlayerInfo.Uuid, saveTokenInfo)
	}

	return reply, nil
}
