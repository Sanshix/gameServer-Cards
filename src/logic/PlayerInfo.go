package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"

	"github.com/golang/protobuf/proto"
)

func init() {
	common.AllComponentMap["PlayerInfo"] = &PlayerInfo{}
}

type PlayerInfo struct {
	base.Base
}

func (p *PlayerInfo) LoadComponent(config *common.OneComponentConfig, componentName string) {
	p.Base.LoadComponent(config, componentName)

	return
}

// LoadPlayerReadOnly 只读操作的读取玩家信息,不加锁
func (p *PlayerInfo) LoadPlayerReadOnly(request *pb.LoadPlayerRequest, extroInfo *pb.MessageExtroInfo) (*pb.LoadPlayerReply, *pb.ErrorMessage) {
	uuid := request.GetUuid()

	reply := &pb.LoadPlayerReply{}

	//先看redis中有没有
	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = common.RedisPlayerInfoTable
	redisRequest.Key = uuid
	redisReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "GetByte", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		//common.LogError("PlayerInfo LoadPlayer Call Redis Get has err", uuid, msgErr)
		return reply, msgErr
	}
	playerInfoByte := redisReply.ValueByte
	if playerInfoByte != nil {
		playerInfo := &pb.PlayerInfo{}
		err := proto.Unmarshal(playerInfoByte, playerInfo)
		if err != nil {
			common.LogError("PlayerInfo LoadPlayer proto.Unmarshal has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		reply.PlayerInfo = playerInfo
		//common.LogInfo("PlayerInfo LoadPlayer from redis ok:", uuid)
		return reply, nil
	}
	//没有就去数据库里读
	mysqlKVRequest := &pb.MysqlKVMessage{}
	mysqlKVRequest.TableName = common.MysqlPlayerInfoTable
	mysqlKVRequest.Uuid = uuid
	mysqlKVReply := &pb.MysqlKVMessage{}
	msgErr = common.Router.Call("Mysql", "QueryKV", mysqlKVRequest, mysqlKVReply, extroInfo)
	if msgErr != nil {
		//common.LogError("PlayerInfo LoadPlayer Call Mysql QueryKV has err", uuid, msgErr)
		return reply, msgErr
	}
	playerInfoByte = mysqlKVReply.GetInfo()
	if playerInfoByte == nil {
		common.LogError("PlayerInfo LoadPlayer playerInfoByte == nil in mysql", uuid)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	playerInfo := &pb.PlayerInfo{}
	err := proto.Unmarshal(playerInfoByte, playerInfo)
	if err != nil {
		common.LogError("PlayerInfo LoadPlayer proto.Unmarshal after mysql has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	if playerInfo.Status == pb.UserStatus_invalid {
		playerInfo.Status = pb.UserStatus_normal
	}
	reply.PlayerInfo = playerInfo
	//common.LogInfo("PlayerInfo LoadPlayer from mysql ok:", uuid)
	return reply, nil
}

func (p *PlayerInfo) LoadPlayer(request *pb.LoadPlayerRequest, extroInfo *pb.MessageExtroInfo) (*pb.LoadPlayerReply, *pb.ErrorMessage) {
	uuid := request.GetUuid()

	playerMutex, err := p.ComponentLock(common.MessageLockPlayer+uuid, extroInfo)
	if err != nil {
		common.LogError("PlayerInfo LoadPlayer  MessageLockPlayer has err", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer p.ComponentUnlock(common.MessageLockPlayer+uuid, extroInfo, playerMutex)

	//common.LogDebug("PlayerInfo LoadPlayer extroInfo", extroInfo)
	//common.LogDebug("PlayerInfo LoadPlayer request", request)

	reply := &pb.LoadPlayerReply{}

	//先看redis中有没有
	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = common.RedisPlayerInfoTable
	redisRequest.Key = uuid
	redisReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "GetByte", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		//common.LogError("PlayerInfo LoadPlayer Call Redis Get has err", uuid, msgErr)
		return reply, msgErr
	}
	playerInfoByte := redisReply.ValueByte
	if playerInfoByte != nil {
		playerInfo := &pb.PlayerInfo{}
		err := proto.Unmarshal(playerInfoByte, playerInfo)
		if err != nil {
			common.LogError("PlayerInfo LoadPlayer proto.Unmarshal has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		reply.PlayerInfo = playerInfo
		//common.LogInfo("PlayerInfo LoadPlayer from redis ok:", uuid)
		return reply, nil
	}
	//没有就去数据库里读
	mysqlKVRequest := &pb.MysqlKVMessage{}
	mysqlKVRequest.TableName = common.MysqlPlayerInfoTable
	mysqlKVRequest.Uuid = uuid
	mysqlKVReply := &pb.MysqlKVMessage{}
	msgErr = common.Router.Call("Mysql", "QueryKV", mysqlKVRequest, mysqlKVReply, extroInfo)
	if msgErr != nil {
		//common.LogError("PlayerInfo LoadPlayer Call Mysql QueryKV has err", uuid, msgErr)
		return reply, msgErr
	}
	playerInfoByte = mysqlKVReply.GetInfo()
	if playerInfoByte == nil {
		common.LogError("PlayerInfo LoadPlayer playerInfoByte == nil in mysql", uuid)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	playerInfo := &pb.PlayerInfo{}
	err = proto.Unmarshal(playerInfoByte, playerInfo)
	if err != nil {
		common.LogError("PlayerInfo LoadPlayer proto.Unmarshal after mysql has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//有就存到redis里一份
	redisRequest = &pb.RedisMessage{}
	redisRequest.Table = common.RedisPlayerInfoTable
	redisRequest.Key = uuid
	redisRequest.ValueByte = playerInfoByte
	redisReply = &pb.RedisMessage{}
	msgErr = common.Router.Call("Redis", "SetByte", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		//common.LogError("PlayerInfo LoadPlayer Call Redis Set has err", msgErr)
		return reply, msgErr
	}
	if playerInfo.Status == pb.UserStatus_invalid {
		playerInfo.Status = pb.UserStatus_normal
	}
	reply.PlayerInfo = playerInfo
	//common.LogInfo("PlayerInfo LoadPlayer from mysql ok:", uuid)
	return reply, nil
}

func (p *PlayerInfo) NewPlayer(request *pb.NewPlayerRequest, extroInfo *pb.MessageExtroInfo) (*pb.NewPlayerReply, *pb.ErrorMessage) {
	uuid := request.GetUuid()
	shortId := request.GetShortId()
	roleType := request.GetRoleType()

	playerMutex, err := p.ComponentLock(common.MessageLockPlayer+uuid, extroInfo)
	if err != nil {
		common.LogError("PlayerInfo LoadPlayer  MessageLockPlayer has err", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer p.ComponentUnlock(common.MessageLockPlayer+uuid, extroInfo, playerMutex)

	//common.LogDebug("PlayerInfo NewPlayer extroInfo", extroInfo)
	//common.LogDebug("PlayerInfo NewPlayer request", request)

	reply := &pb.NewPlayerReply{}

	playerInfo := &pb.PlayerInfo{}
	playerInfo.Name = request.Name
	playerInfo.Balance = 1000000
	playerInfo.Uuid = uuid
	playerInfo.ShortId = shortId
	playerInfo.Auths = request.Auths
	playerInfo.Status = pb.UserStatus_normal
	playerInfo.Role = roleType
	playerInfo.Account = request.GetAccount()

	playerInfo.Sex = request.Sex
	playerInfo.ThirdPartyAccessInfo = request.GetThirdPartyAccessInfo()
	playerInfo.Province = request.GetProvince()
	playerInfo.City = request.GetCity()
	playerInfo.HeadImgUrl = request.GetHeadImgUrl()
	playerInfo.Country = request.GetCountry()
	playerInfo.PlayerSourceType = request.GetPlayerSourceType()

	if roleType == pb.Roles_Robot {
		playerInfo.IsRobot = true
	}

	playerInfoNone := &pb.PlayerInfo{}
	//注册，保存一份排行榜相关
	if common.SelectComponentExist("LeaderBoard") { // 当有这个组件的时候

		msgErr := common.Router.Call("LeaderBoard", "UpdateNabobLeaderBoard", playerInfo, playerInfoNone, extroInfo)
		if msgErr != nil {
			common.LogError("PlayerInfo NewPlayer call LeaderBoard UpdateNabobLeaderBoard has err", msgErr)
			return reply, msgErr
		}

		msgErr = common.Router.Call("LeaderBoard", "UpdateBigWinnerLeaderBoard", playerInfo, playerInfoNone, extroInfo)
		if msgErr != nil {
			common.LogError("PlayerInfo NewPlayer call LeaderBoard UpdateBigWinnerLeaderBoard has err", msgErr)
			return reply, msgErr
		}
	}

	//第一次登陆时,设置3日的第一次奖励 与 周1礼包可以领
	//登录时 礼包状态判断
	if common.SelectComponentExist("Gift") { // 当有这个组件的时候
		msgErr := common.Router.Call("Gift", "AddGiftForPlayer", playerInfo, playerInfoNone, extroInfo)
		if msgErr != nil {
			common.LogError("PlayerInfo NewPlayer Set Gift has error", err)
			return reply, msgErr
		}
		playerInfo = playerInfoNone
	}

	playerInfoByte, err := proto.Marshal(playerInfo)
	if err != nil {
		common.LogError("PlayerInfo: NewPlayer Marshal has err", uuid, shortId, err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//先存数据库
	mysqlKVRequest := &pb.MysqlKVMessage{}
	mysqlKVRequest.TableName = common.MysqlPlayerInfoTable
	mysqlKVRequest.Uuid = uuid
	mysqlKVRequest.ShortId = shortId
	mysqlKVRequest.Info = playerInfoByte
	mysqlKVReply := &pb.MysqlKVMessage{}
	msgErr := common.Router.Call("Mysql", "InsertKV", mysqlKVRequest, mysqlKVReply, extroInfo)
	if msgErr != nil {
		//common.LogError("PlayerInfo NewPlayer Call Mysql InsertKV has err", uuid, shortId, msgErr)
		return reply, msgErr
	}
	//再存redis
	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = common.RedisPlayerInfoTable
	redisRequest.Key = uuid
	redisRequest.ValueByte = playerInfoByte
	redisReply := &pb.RedisMessage{}
	msgErr = common.Router.Call("Redis", "SetByte", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		//common.LogError("PlayerInfo NewPlayer Call Redis Set has err", msgErr)
		return reply, msgErr
	}

	reply.PlayerInfo = playerInfo
	//common.LogInfo("PlayerInfo NewPlayer ok:", uuid, shortId)
	return reply, nil
}

func (p *PlayerInfo) UpdatePlayer(request *pb.PlayerInfo, extroInfo *pb.MessageExtroInfo) (*pb.PlayerInfo, *pb.ErrorMessage) {

	playerMutex, err := p.ComponentLock(common.MessageLockPlayer+request.Uuid, extroInfo)
	if err != nil {
		common.LogError("PlayerInfo LoadPlayer  MessageLockPlayer has err", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer p.ComponentUnlock(common.MessageLockPlayer+request.Uuid, extroInfo, playerMutex)

	playerInfoInDb, msgErr := p.LoadPlayer(&pb.LoadPlayerRequest{Uuid: request.Uuid}, extroInfo)

	if msgErr != nil {
		return nil, msgErr
	}

	//common.LogDebug("PlayerInfo UpdatePlayer extroInfo", extroInfo)
	//common.LogDebug("PlayerInfo UpdatePlayer request", request)

	playerInfoInDb.PlayerInfo.Name = request.Name
	playerInfoInDb.PlayerInfo.Auths = request.Auths
	playerInfoInDb.PlayerInfo.Status = request.Status
	playerInfoInDb.PlayerInfo.Balance = request.Balance

	savePlayerRequest := &pb.SavePlayerRequest{
		PlayerInfo: playerInfoInDb.PlayerInfo,
		ForceSave:  true,
	}

	_, err1 := p.SavePlayer(savePlayerRequest, extroInfo)
	if err1 != nil {
		return nil, err1
	}
	return playerInfoInDb.PlayerInfo, nil
}

// SavePlayer 保存一次玩家数据
func (obj *PlayerInfo) SavePlayer(request *pb.SavePlayerRequest, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	playerInfo := request.GetPlayerInfo()

	playerMutex, err := obj.ComponentLock(common.MessageLockPlayer+playerInfo.GetUuid(), extroInfo)
	if err != nil {
		common.LogError("PlayerInfo LoadPlayer  MessageLockPlayer has err", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer obj.ComponentUnlock(common.MessageLockPlayer+playerInfo.GetUuid(), extroInfo, playerMutex)

	//common.LogDebug("PlayerInfo SavePlayer extroInfo", extroInfo)
	//common.LogDebug("PlayerInfo SavePlayer request", request)

	reply := &pb.EmptyMessage{}

	playerInfoByte, err := proto.Marshal(playerInfo)
	if err != nil {
		common.LogError("PlayerInfo: SavePlayer Marshal has err", playerInfo.GetUuid(), playerInfo.GetShortId(), err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//存redis
	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = common.RedisPlayerInfoTable
	redisRequest.Key = playerInfo.GetUuid()
	redisRequest.ValueByte = playerInfoByte
	redisReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "SetByte", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		//common.LogError("PlayerInfo NewPlayer Call Redis Set has err", msgErr)
		return reply, msgErr
	}
	if request.GetForceSave() == false {
		return reply, nil
	}
	//存数据库
	mysqlKVRequest := &pb.MysqlKVMessage{}
	mysqlKVRequest.TableName = common.MysqlPlayerInfoTable
	mysqlKVRequest.Uuid = playerInfo.GetUuid()
	mysqlKVRequest.ShortId = playerInfo.GetShortId()
	mysqlKVRequest.Info = playerInfoByte
	mysqlKVReply := &pb.MysqlKVMessage{}
	msgErr = common.Router.Call("Mysql", "UpdateKV", mysqlKVRequest, mysqlKVReply, extroInfo)
	if msgErr != nil {
		//common.LogError("PlayerInfo NewPlayer Call Mysql InsertKV has err", uuid, shortId, msgErr)
		return reply, msgErr
	}

	//common.LogInfo("PlayerInfo SavePlayer ok:", playerInfo.GetUuid(), playerInfo.GetShortId())
	return reply, nil
}

func (obj *PlayerInfo) DeletePlayer(request *pb.LoadPlayerRequest, extroInfo *pb.MessageExtroInfo) (*pb.LoadPlayerReply, *pb.ErrorMessage) {
	uuid := request.GetUuid()
	reply := &pb.LoadPlayerReply{}
	playerMutex, err := obj.ComponentLock(common.MessageLockPlayer+uuid, extroInfo)
	if err != nil {
		common.LogError("PlayerInfo DeletePlayer  MessageLockPlayer has err", uuid, err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer obj.ComponentUnlock(common.MessageLockPlayer+uuid, extroInfo, playerMutex)

	mysqlDeleteRequest := &pb.MysqlAccountInfo{}
	mysqlDeleteRequest.Uuid = uuid
	mysqlDeleteReply := &pb.MysqlAccountInfo{}
	deleteMsgErr := common.Router.Call("Mysql", "ForceDeleteAccount", mysqlDeleteRequest, mysqlDeleteReply, extroInfo)
	if deleteMsgErr != nil {
		common.LogError("PlayerInfo DeletePlayer Call Mysql ForceDeleteAccount has err", mysqlDeleteRequest, uuid, deleteMsgErr)
	}

	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = common.RedisPlayerInfoTable
	redisRequest.Key = uuid
	redisReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "Delete", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		common.LogError("PlayerInfo DeletePlayer Call Redis Delete has err", uuid, msgErr)
	}

	mysqlKVRequest := &pb.MysqlKVMessage{}
	mysqlKVRequest.TableName = common.MysqlPlayerInfoTable
	mysqlKVRequest.Uuid = uuid
	mysqlKVReply := &pb.MysqlKVMessage{}
	msgErr = common.Router.Call("Mysql", "DeleteKV", mysqlKVRequest, mysqlKVReply, extroInfo)
	if msgErr != nil {
		common.LogError("PlayerInfo DeletePlayer Call Mysql DeleteKV has err", uuid, msgErr)
	}
	return reply, nil
}
