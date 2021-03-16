package common

import (
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

// GameReadyFunc 房间准备函数
// 参数：房间信息
// 返回值：rpc错误
type GameReadyFunc func(roomInfo *pb.RoomInfo) *pb.ErrorMessage

// GameDriverSyncRoomPlayerInfo 同步玩家信息
// 参数：同步信息，房间管理器，rpc附加消息
// 返回：房间信息，rpc错误信息
func GameDriverSyncRoomPlayerInfo(request *pb.SyncRoomPlayerInfo, rm *RoomManager, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage {
	return rm.SyncRoomPlayerInfo(request)
}

// GameDriverDo 房间操作通用接口，由driver向逻辑层通信用
// 参数：逻辑组件名，逻辑方法名房间管理器，请求信息，房间管理器，rpc附加消息
// 返回：回执消息，rpc错误信息
func GameDriverDo(componentName string, methodName string, request proto.Message, reply proto.Message, rm *RoomManager, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage {

	uuid := extroInfo.GetUserId()
	if uuid == "" {
		LogError("GameDriverExitRoom uuid == nil")
		return GetGrpcErrorMessage(pb.ErrorCode_UserNotLogin, "")
	}

	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		return msgErr
	}

	_, msgErr = rm.Do(loadPlayerReply.GetPlayerInfo(), func(roomInfo *pb.RoomInfo) (*pb.RoomInfo, proto.Message, *pb.ErrorMessage) {

		driver2LogicRequest := &pb.Driver2GameLogicInfo{}
		driver2LogicRequest.RoomInfo = roomInfo
		requestMessageAny, err := ptypes.MarshalAny(request)
		if err != nil {
			LogError("GameDriverDo MarshalAny has err", err)
			return nil, nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		driver2LogicRequest.Message = requestMessageAny
		driver2LogicReply := &pb.Driver2GameLogicInfo{}

		msgErr := Router.Call(componentName, methodName, driver2LogicRequest, driver2LogicReply, extroInfo)
		if msgErr != nil {
			return nil, nil, msgErr
		}
		err = ptypes.UnmarshalAny(driver2LogicReply.GetMessage(), reply)
		if err != nil {
			LogError("GameDriverDo ptypes.UnmarshalAny has err", err)
			return nil, nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		afterRoomInfo := driver2LogicReply.GetRoomInfo()

		return afterRoomInfo, reply, msgErr
	})

	return msgErr
}

// HundredGameReadyDiver 百人场游戏准备diver驱动公用
//参数：request房间详细,readyFunc回调
//返回：房间信息 , 报错
func HundredGameReadyDiver(request *pb.RoomInfo, readyFunc GameReadyFunc) (*pb.RoomInfo, *pb.ErrorMessage) {
	return HundredGameReadyDiverNextState(request, readyFunc, pb.RoomState_RoomStateBankChange)
}

// HundredGameReadyDiverNextState 指定下个状态的百人场游戏准备diver驱动公用
//参数：request房间详细,readyFunc回调,nextState指定的下个状态
//返回：房间信息 , 报错
func HundredGameReadyDiverNextState(request *pb.RoomInfo, readyFunc GameReadyFunc, nextState pb.RoomState) (*pb.RoomInfo, *pb.ErrorMessage) {
	//1.准备阶段刷新房间配置
	gameKeyMap := Configer.GetGameConfigByGameTypeAndScene(request.GetGameType(), request.GetGameScene())
	if gameKeyMap != nil {
		request.Config = []*pb.GameConfig{}
		for _, oneConfig := range gameKeyMap.Map {
			request.Config = append(request.Config, oneConfig)
		}
	}

	nowTime := time.Now().Unix()
	// 当下个状态不是状态状态时，准备<->庄家改变
	if request.GetNextRoomState() != pb.RoomState_RoomStateReady {
		if nowTime < request.DoTime {
			return request, nil
		}
		request.CurRoomState = nextState
		request.NextRoomState = nextState
		request.DoTime = nowTime
		return request, nil
	}

	// 记录游戏开始时间
	request.RoundStartTime = nowTime

	//2.根据游戏类型获取开始时间
	readyTimeStr := GetRoomConfig(request, "ReadyTime")
	readyTime, err := strconv.Atoi(readyTimeStr)
	if err != nil {
		LogError("roomReadyLogic HundredGameReadyDiver  readyTimeStr has err", err)
		return request, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//3.清除房间里面不在线的玩家 + 初始化玩家状态
	for v, k := range request.PlayerInfo {
		//获取对应用户在线状态
		isOnline, msgErr := Pusher.CheckOnline(k.GetUuid())
		if msgErr != nil {
			LogError("CheckOnline has err", k.GetUuid(), msgErr)
			isOnline = false
		}
		// 初始化玩家下注记录、输赢信息、奖池获取
		switch request.GameType {
		case pb.GameType_DragonTigerFight:
			request.PlayerInfo[v].PlayerBets = make([]int64, 3)
		case pb.GameType_PushBobbin:
			request.PlayerInfo[v].PlayerBets = make([]int64, 3)
		case pb.GameType_HundredBull:
			request.PlayerInfo[v].PlayerBets = make([]int64, 8)
		case pb.GameType_RedBlack:
			request.PlayerInfo[v].PlayerBets = make([]int64, 3)
		}
		request.PlayerInfo[v].WinOrLose = 0
		request.PlayerInfo[v].GetBonus = 0
		request.PlayerInfo[v].HundredWaterBill = 0
		request.PlayerInfo[v].HundredBonusTax = 0
		request.PlayerInfo[v].HundredCommission = 0
		// 龙虎特有
		request.PlayerInfo[v].DragonTigerWinMoney = 0
		// 红黑特有
		request.PlayerInfo[v].RedBlackOnlyWinMoney = 0
		// 当查在线表后，有他时就跳过，没有就标记将他踢掉
		if isOnline {
			// 复原玩家状态
			if k.Uuid == request.BankerUuid || PlayerIsInBankers(k.Uuid, request) {
				continue
			}
			request.PlayerInfo[v].PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateFree
			continue
		}
		request.PlayerInfo[v].PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateNone
		request.PlayerInfo[v].WaitKick = pb.RoomSeatsChangeReason_RoomSeatsChangeReason_KickDisconnect
	}

	// 4.给房间所有人推送状态改变的消息
	// 这里前一个状态就是前一个状态，下一个状态就是当前正处在的状态
	// 结算 < -- > 准备
	pushRoomState := &pb.PushRoomStateChange{
		RoomId:            request.GetUuid(),
		BeforeState:       pb.RoomState_RoomStateSettle,
		AfterState:        pb.RoomState_RoomStateReady,
		AfterStateEndTime: nowTime + int64(readyTime),
	}
	RoomBroadcast(request, pushRoomState)

	// 5.初始化房间状态
	request.MaxBetRatio = make([]int64, 3)
	request.CurrentRoundId = uuid.NewV4().String()
	// 龙虎
	request.DragonTigerAllBet = make([]int64, 3)
	request.DragonTigerPoker = nil
	// 红黑
	request.RedBlackPokerType = nil
	request.RedBlackPoker = nil
	request.RedBlackAllBet = make([]int64, 3)
	// 百人牛牛
	request.HundredBullAllBet = make([]int64, 8)
	request.HundredBullPokerList = []*pb.HundredBullPokerCard{}
	// 推筒子
	request.AllBet = make([]int64, 4)
	request.PushBobbinMahjongList = []*pb.PushBobbinMahjong{}

	//金币房每次开始的时候需要清空上一局结算信息
	if GameMode == pb.GameMode_GameMode_Gold {
		request.AllSettleInfo = []*pb.SettleInfo{}
	}

	// 更新下个状态
	request.NextRoomState = nextState

	request.DoTime = nowTime + int64(readyTime)

	// 调用各自游戏私有逻辑
	if readyFunc != nil {
		msgErr := readyFunc(request)
		if msgErr != nil {
			return request, msgErr
		}
	}

	return request, nil
}

// 公用抢座模式逻辑
//参数：request房间详细
//返回：房间信息 , 报错
//func RoomGardReadyLogic(request *pb.RoomInfo) (*pb.RoomInfo, *pb.ErrorMessage) {
//	// 准备阶段刷新房间配置 -- 实时
//	gameKeyMap := Configer.GetGameConfigByGameTypeAndScene(request.GetGameType(), request.GetGameScene())
//	if gameKeyMap != nil {
//		request.Config = []*pb.GameConfig{}
//		for _, oneConfig := range gameKeyMap.Map {
//			request.Config = append(request.Config, oneConfig)
//		}
//	}
//
//	// 如果桌子信息为空 -- 即第一次进入,创建桌子
//	if request.Table == nil || len(request.Table) != 8 {
//		LogDebug("RoomGardReadyLogic the room first Start uuid =", request.GetUuid(), len(request.Table))
//		request.Table = make([]string, 8)
//	}
//
//	// 获取上座金额限制
//	upSeatStr := GetRoomConfig(request, "UpSeat")
//	upSeat, err := strconv.Atoi(upSeatStr)
//	if err != nil {
//		LogError("RoomGardReadyLogic Drive upSeatStr has err", err)
//		return request, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
//	}
//
//	// 金额判断 + 踢人
//	for index, onePlayer := range request.GetPlayerInfo() {
//
//		if onePlayer.GetUuid() != "" {
//			// 玩家上座，但钱不够/申请了下座
//			if request.PlayerInfo[index].IsSeat && (request.PlayerInfo[index].Balance < int64(upSeat) || request.PlayerInfo[index].DownSeatRequest) {
//				onePlayerTableIndex := GetTableIndex(onePlayer.Uuid, request)
//				if onePlayerTableIndex != -1 {
//					request.PlayerInfo[index].PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateFree
//					request.PlayerInfo[index].IsSeat = false
//					request.Table[onePlayerTableIndex] = ""
//					pushMsg := &pb.PushTableList{
//						RoomId: request.Uuid,
//						Table:  request.Table,
//					}
//					RoomBroadcast(request, pushMsg)
//				} else { // 只是一个错误的情况，打印出来
//					LogError("RoomGardReadyLogic Drive downSeat has err: table not this player,", request.Table[onePlayerTableIndex], onePlayer)
//				}
//			}
//
//			// 离线判断
//			isOnline, msgErr := Pusher.CheckOnline(onePlayer.GetUuid())
//			if msgErr != nil {
//				LogError("RoomGardReadyLogic Drive CheckOnline has err", onePlayer.GetUuid(), msgErr)
//				isOnline = false
//			}
//			if isOnline == false {
//				onePlayerTableIndex := GetTableIndex(onePlayer.Uuid, request)
//
//				if onePlayerTableIndex != -1 {
//					request.PlayerInfo[index].IsSeat = false
//					request.Table[onePlayerTableIndex] = ""
//					pushMsg := &pb.PushTableList{
//						RoomId: request.Uuid,
//						Table:  request.Table,
//					}
//					RoomBroadcast(request, pushMsg)
//				}
//
//				onePlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateNone
//				onePlayer.WaitKick = pb.RoomSeatsChangeReason_RoomSeatsChangeReason_KickDisconnect
//			}
//		}
//
//	}
//
//	nowTime := time.Now().Unix()
//
//	// 获取游戏开始的人数
//	playerStartNumStr := GetRoomConfig(request, "PlayerStartNum")
//	playerStartNum, err := strconv.Atoi(playerStartNumStr)
//	if err != nil {
//		LogError("RoomGardReadyLogic Drive playerStartNumStr has err", playerStartNumStr)
//		return request, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
//	}
//
//	// 当下个阶段不是准备时 判断 - 不用根据DoTime判断 - 直接可开始
//	if request.NextRoomState != pb.RoomState_RoomStateReady {
//
//		// 当 上座/准备 人数 >= playerStartNum[打旋4] 直接开始
//		if GetReadyPlayerNum(request) >= playerStartNum {
//			for _, onePlayer := range request.PlayerInfo {
//				if onePlayer.GetUuid() == "" {
//					continue
//				}
//				for _, onePlayer := range request.GetPlayerInfo() {
//					if onePlayer.GetPlayerRoomState() == pb.PlayerRoomState_PlayerRoomStateReady {
//						onePlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStatePlay
//					} else {
//						onePlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateWatch
//					}
//				}
//			}
//			// 推送
//			pushMsg := &pb.PushRoomStateChange{
//				RoomId:            request.Uuid,
//				BeforeState:       pb.RoomState_RoomStateReady,
//				AfterState:        pb.RoomState_RoomStateDeal,
//				AfterStateEndTime: nowTime,
//			}
//			RoomBroadcast(request, pushMsg)
//			request.RoundStartTime = time.Now().Unix()
//			request.CurrentRoundId = uuid.NewV4().String()
//			request.CurRoomState = pb.RoomState_RoomStateDeal
//			request.NextRoomState = pb.RoomState_RoomStateDeal
//			request.DoTime = nowTime
//			return request, nil
//		}
//
//		request.DoTime = nowTime + 1
//		return request, nil
//
//	}
//
//	// 第一次进入 清理房间/玩家数据
//	for _, onePlayer := range request.PlayerInfo {
//		if onePlayer.Uuid != "" {
//			// 上座的玩家 状态 -> 准备
//			// 非上座玩家 状态 -> 空闲
//			if onePlayer.IsSeat {
//				onePlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateReady
//
//			} else {
//				onePlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateFree
//			}
//		}
//	}
//
//	// 准备阶段 每1s驱动1次
//	request.DoTime = nowTime + 1
//	request.NextRoomState = pb.RoomState_RoomStateDeal
//	return request, nil
//}
