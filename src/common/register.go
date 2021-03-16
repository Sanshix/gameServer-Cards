package common

import pb "gameServer-demo/src/grpc"

// Register 注册公共流程
func Register(account string, password string, mobile string, role pb.Roles, componentName string, extroInfo *pb.MessageExtroInfo, callback func(*pb.MysqlAccountInfo) *pb.ErrorMessage) (*pb.RegisterReply, *pb.ErrorMessage) {
	reply := &pb.RegisterReply{}
	getShortIDReq := &pb.GetShortIdRequest{}
	getShortIDReply := &pb.GetShortIdReply{}
	shortidErr := Router.Call("Shortid", "Get", getShortIDReq, getShortIDReply, extroInfo)
	if shortidErr != nil {
		LogError("Register Register Call Shortid Get has err", account, shortidErr)
		return reply, shortidErr
	}

	playerMutex, err := Locker.MessageLock(MessageLockAccountRegister+account, extroInfo, componentName)
	if err != nil {
		LogError("PlayerInfo LoadPlayer  MessageLockPlayer has err", err)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockAccountRegister+account, extroInfo, componentName, playerMutex)

	LogDebug("Register Register extroInfo", extroInfo)
	//LogDebug("Register Register request", request)

	mysqlRequest := &pb.MysqlAccountInfo{}
	mysqlRequest.Account = account
	mysqlRequest.Password = password
	mysqlRequest.ShortId = getShortIDReply.GetShortId()
	mysqlRequest.RoleType = role
	mysqlRequest.Mobile = mobile
	mysqlRequest.PlayerSourceType = pb.PlayerSourceType_PlayerSourceType_Local
	mysqlReply := &pb.MysqlAccountInfo{}
	msgErr := Router.Call("Mysql", "NewAccount", mysqlRequest, mysqlReply, extroInfo)
	if msgErr != nil {
		LogError("Register Register Call Mysql NewAccount has err", account, msgErr)
		return reply, msgErr
	}
	newPlayerRequest := &pb.NewPlayerRequest{}
	newPlayerRequest.Uuid = mysqlReply.GetUuid()
	newPlayerRequest.ShortId = mysqlReply.GetShortId()
	newPlayerRequest.RoleType = role
	newPlayerRequest.Account = mysqlReply.GetAccount()
	newPlayerRequest.Mobile = mobile
	newPlayerReply := &pb.NewPlayerReply{}
	msgErr = Router.Call("PlayerInfo", "NewPlayer", newPlayerRequest, newPlayerReply, extroInfo)
	if msgErr != nil {
		LogError("Register Register Call PlayerInfo NewPlayer has err", msgErr)
		return reply, msgErr
	}
	if callback != nil {
		msgErr := callback(mysqlReply)
		if msgErr != nil {
			// 删除之前创建的账号
			LogError("Register Register callback has err", account, mysqlReply, msgErr)
			/*mysqlDeleteRequest := &pb.MysqlAccountInfo{}
			mysqlDeleteRequest.Uuid = mysqlReply.GetUuid()
			mysqlDeleteReply := &pb.MysqlAccountInfo{}
			deleteMsgErr := Router.Call("Mysql", "ForceDeleteAccount", mysqlDeleteRequest, mysqlDeleteReply, extroInfo)
			if deleteMsgErr != nil {
				LogError("Register Register Call Mysql ForceDeleteAccount has err", mysqlDeleteRequest, deleteMsgErr)
			}*/
			loadPlayerRequest := &pb.LoadPlayerRequest{}
			loadPlayerRequest.Uuid = mysqlReply.GetUuid()
			loadPlayerReply := &pb.LoadPlayerReply{}
			msgErr = Router.Call("PlayerInfo", "DeletePlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
			if msgErr != nil {
				LogError("Register Register Call PlayerInfo DeletePlayers has err", msgErr)
				return reply, msgErr
			}
			return reply, msgErr
		}
	}
	reply.Uuid = mysqlReply.GetUuid()
	reply.ShortId = mysqlReply.GetShortId()
	LogDebug("Register Register ok", mysqlReply)
	return reply, nil
}
