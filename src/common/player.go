package common

import (
	pb "gameServer-demo/src/grpc"
	"strconv"
)

// HandleOtherPlayerFunc 加锁操作某个玩家的信息的回掉，外部已加锁
// 返回true则进行入库操作，一般重要的信息修改，如修改金钱则需要返回true，其他返回false
type HandleOtherPlayerFunc func(playerInfo *pb.PlayerInfo, extraInfo *pb.MessageExtroInfo) (bool, *pb.ErrorMessage)

// HandleOtherPlayerInfo 加锁操作某个玩家的信息
// 当你拿不到extroInfo，又要去操作玩家信息时用这个方法，该方法会加锁执行外部传入的逻辑
// 当你在客户端上行协议中时请勿使用该方法，上行时自己已经有加锁了
// 无论在什么时候使用该方法，请考虑锁嵌套的问题，尽量使用协程来执行此方法
func HandleOtherPlayerInfo(uuid string, handleFunc HandleOtherPlayerFunc) *pb.ErrorMessage {
	extroInfo := &pb.MessageExtroInfo{}
	componentName := "HandleOtherPlayerInfo"
	playerMutex, err := Locker.MessageLock(MessageLockPlayer+uuid, extroInfo, componentName)
	if err != nil {
		LogError("HandleOtherPlayerInfo  MessageLockPlayer has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockPlayer+uuid, extroInfo, componentName, playerMutex)

	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		return msgErr
	}
	playerInfo := loadPlayerReply.GetPlayerInfo()
	isForceSave, msgErr := handleFunc(playerInfo, extroInfo)
	if msgErr != nil {
		return msgErr
	}
	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = isForceSave
	msgErr = Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
	if msgErr != nil {
		return msgErr
	}
	return nil
}

// ChangePlayerInfoAfterGameEnd 游戏结束后加锁修改玩家信息
// 当你拿不到extroInfo，又要去操作玩家信息时用这个方法
//
// 这个方法适用于被修改玩家是在房间中的，不需要再同步一次房间内信息的方法
//
// 当你在客户端上行协议中时请勿使用该方法，上行时自己已经有加锁了
// 无论在什么时候使用该方法，请考虑锁嵌套的问题，尽量使用协程来执行此方法
//
// taskConfig触发任务所需要的一些参数，暂时这样做，后期做记录时再修改
// changeBalance游戏的输赢
// 返回修改后的金额
func ChangePlayerInfoAfterGameEnd(uuid string, changeBalance int64, taskConfig *pb.TaskConfig, changeReason pb.ResourceChangeReason) (int64, *pb.ErrorMessage) {
	afterBalance := int64(0)
	msgErr := HandleOtherPlayerInfo(uuid, func(playerInfo *pb.PlayerInfo, extraInfo *pb.MessageExtroInfo) (bool, *pb.ErrorMessage) {
		msgErr1 := AddResource(pb.RewardType_Golden, changeBalance, playerInfo, false, extraInfo, true, changeReason)
		afterBalance = playerInfo.GetBalance()
		return true, msgErr1
	})
	return afterBalance, msgErr
}

// ChangeOtherBalance 加锁修改别的玩家的金钱
// 当你拿不到extroInfo，又要去操作其他玩家信息时用这个方法
//
// 这个方法适用修改他人金钱
//
// 无论在什么时候使用该方法，请考虑锁嵌套的问题，尽量使用协程来执行此方法
//
// isSyncRoom 是否要同步到房间,如果在房间中就不需要
// changeBalance游戏的输赢
// 返回修改后的金额
func ChangeOtherBalance(uuid string, changeBalance int64, isSyncRoom bool, isForce bool, changeReason pb.ResourceChangeReason) *pb.ErrorMessage {
	msgErr := HandleOtherPlayerInfo(uuid, func(playerInfo *pb.PlayerInfo, extraInfo *pb.MessageExtroInfo) (bool, *pb.ErrorMessage) {
		msgErr1 := AddResource(pb.RewardType_Golden, changeBalance, playerInfo, isSyncRoom, extraInfo, isForce, changeReason)
		return true, msgErr1
	})
	return msgErr
}

// AddResource 不加锁-修改玩家获取的资源-到玩家信息里面
// isForce 扣除资源时，如果是true，则有多少扣多少，否则不够就返回错误
// 资源修改加推送 /*暂不推送*/
func AddResource(reSource pb.RewardType, num int64, playerInfo *pb.PlayerInfo, isSyncRoom bool, extraInfo *pb.MessageExtroInfo, isForce bool, reason pb.ResourceChangeReason) *pb.ErrorMessage {
	switch reSource {
	case pb.RewardType_Golden:
		// 如果需要同步到房间，则先修改房间数据
		if isSyncRoom == true {
			if playerInfo.GetRoomId() != "" {
				reply := &pb.EmptyMessage{}
				request := &pb.SyncRoomPlayerInfo{}
				request.ChangeBalance = num
				request.PlayerUUID = playerInfo.GetUuid()
				request.RoomUUID = playerInfo.GetRoomId()
				request.ServerIndex = playerInfo.GetGameServerIndex()
				gameType := playerInfo.GetGameType().String()
				msgErr := Router.Call(gameType+"Route", "SyncRoomPlayerInfo", request, reply, extraInfo)
				if msgErr != nil {
					LogError("AddResource isSyncRoom == true SyncRoomPlayerInfo has err", msgErr)
					return msgErr
				}
			}
		}
		playerInfo.Balance += num
		if isForce == true {
			if playerInfo.Balance < 0 {
				playerInfo.Balance = 0
			}
		} else {
			if playerInfo.Balance < 0 {
				LogError("AddResource isForce == false playerInfo.Balance < 0")
				return GetGrpcErrorMessage(pb.ErrorCode_BalanceNotEnough, "")
			}
		}
		// 金币变动原因是游戏分段时
		if reason >= 1000 && reason <= 3000 {
			if num > 0 {
				playerInfo.WinMoneySum += num
			}
		}

		// 跑马灯赢钱推送
		go func() {
			if SelectComponentExist("HorseRaceLamp") {
				if reason >= 1000 && reason <= 3000 {
					hInt, err := strconv.Atoi(Configer.GetGlobal("HorseRaceLampWin").GetValue())
					if err != nil {
						LogError("strconv.Atoi(oddsStr) has err", err)
					}
					if num >= int64(hInt) {
						horseRaceLampWinerRequest := &pb.HorseRaceLampWinPlayer{}
						horseRaceLampWinerRequest.PlayerInfos = playerInfo
						horseRaceLampWinerRequest.Num = num
						horseRaceLampReply := &pb.HorseRaceLampReply{}
						err := Router.Call("HorseRaceLamp", "PushHorseRaceLampWiner", horseRaceLampWinerRequest, horseRaceLampReply, &pb.MessageExtroInfo{})
						if err != nil {
							LogError("Player PushHorseRaceLampWiner call  has err", err)
						}
					}
				}
			}
		}()

		if num != 0 {
			//推送金额变动报表通知
			go PushPlayerBalanceChangeRecord(playerInfo, num, reason)
		}

	case pb.RewardType_SpinNum:
		playerInfo.SpinNum = playerInfo.GetSpinNum() + int32(num)
	}

	//保存到排行榜相关 - 如组件存在
	go func() {
		if SelectComponentExist("LeaderBoard") {
			playerInfoNone := &pb.PlayerInfo{}
			msgErr := Router.Call("LeaderBoard", "UpdateNabobLeaderBoard", playerInfo, playerInfoNone, extraInfo)
			if msgErr != nil {
				LogError("Player AddResource call LeaderBoard UpdateNabobLeaderBoard has err", msgErr)
			}
			msgErr = Router.Call("LeaderBoard", "UpdateBigWinnerLeaderBoard", playerInfo, playerInfoNone, extraInfo)
			if msgErr != nil {
				LogError("Player AddResource call LeaderBoard UpdateBigWinnerLeaderBoard has err", msgErr)
			}
		}
	}()

	//推送
	userResourceChangePush := &pb.PushUserBalanceChange{}
	userResourceChangePush.UserId = playerInfo.GetUuid()
	userResourceChangePush.Balance = playerInfo.Balance
	Pusher.Push(userResourceChangePush, playerInfo.GetUuid())
	return nil
}

// GetAccountInfoByShortID 通过短位id查询账号信息
func GetAccountInfoByShortID(shortID string) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	mysqlRequest := &pb.MysqlAccountInfo{}
	mysqlRequest.ShortId = shortID
	mysqlReply := &pb.MysqlAccountInfo{}
	msgErr := Router.Call("Mysql", "QueryAccountByShortID", mysqlRequest, mysqlReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("GetAccountInfoByShortID Call Mysql QueryAccountByShortID has err", shortID, msgErr)
		return mysqlReply, msgErr
	}
	return mysqlReply, nil
}

// GetAccountInfoByUUID 通过uuid查询账号信息
func GetAccountInfoByUUID(UUID string) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	mysqlRequest := &pb.MysqlAccountInfo{}
	mysqlRequest.Uuid = UUID
	mysqlReply := &pb.MysqlAccountInfo{}
	msgErr := Router.Call("Mysql", "QueryAccountByUUID", mysqlRequest, mysqlReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("GetAccountInfoByUUID Call Mysql QueryAccountByUUID has err", UUID, msgErr)
		return mysqlReply, msgErr
	}
	return mysqlReply, nil
}

// GetAccountInfoByAccount 通过account查询账号信息
func GetAccountInfoByAccount(account string) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	mysqlRequest := &pb.MysqlAccountInfo{}
	mysqlRequest.Account = account
	mysqlReply := &pb.MysqlAccountInfo{}
	msgErr := Router.Call("Mysql", "QueryAccountByAccount", mysqlRequest, mysqlReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("GetAccountInfoByUUID Call Mysql QueryAccountByAccount has err", account, msgErr)
		return mysqlReply, msgErr
	}
	return mysqlReply, nil
}

// GetAccountInfoByMobile 通过mobile查询账号信息
func GetAccountInfoByMobile(mobile string) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	mysqlRequest := &pb.MysqlAccountInfo{}
	mysqlRequest.Mobile = mobile
	mysqlReply := &pb.MysqlAccountInfo{}
	msgErr := Router.Call("Mysql", "QueryAccountByMobile", mysqlRequest, mysqlReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("GetAccountInfoByUUID Call Mysql QueryAccountByMobile has err", mobile, msgErr)
		return mysqlReply, msgErr
	}
	return mysqlReply, nil
}
