package common

import (
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"
)

// GameSettleFunc 房间结算函数
// 参数：房间信息
// 返回值：rpc错误
type GameSettleFunc func(roomInfo *pb.RoomInfo) *pb.ErrorMessage

// HundredGameSettleDiver 百人场游戏结算diver公用
// 参数:request，驱动启动时间nowTime，游戏单独的逻辑
// 返回值：房间信息,错误信息
func HundredGameSettleDiver(request *pb.RoomInfo, settleFunc GameSettleFunc) (*pb.RoomInfo, *pb.ErrorMessage) {
	// 获取当前时间时间戳
	nowTime := time.Now().Unix()

	// 当下个状态不是状态状态时， 结算 <-> 准备
	if request.GetNextRoomState() != pb.RoomState_RoomStateSettle {
		if nowTime < request.DoTime {
			return request, nil
		}
		request.CurRoomState = pb.RoomState_RoomStateReady
		request.NextRoomState = pb.RoomState_RoomStateReady
		request.DoTime = nowTime
		return request, nil
	}

	// 获取结算时间
	settleTimeStr := GetRoomConfig(request, "SettleTime")
	settleTime, err := strconv.Atoi(settleTimeStr)
	if err != nil {
		LogError("roomSettleLogic HundredGameSettleDiver settleTimeStr has err", err)
		return request, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 推送房间状态 发牌<->结算
	pushRoomState := &pb.PushRoomStateChange{
		RoomId:            request.GetUuid(),
		BeforeState:       pb.RoomState_RoomStateDeal,
		AfterState:        pb.RoomState_RoomStateSettle,
		AfterStateEndTime: nowTime + int64(settleTime),
	}
	RoomBroadcast(request, pushRoomState)

	// 执行各游戏自己的逻辑
	msgErr := settleFunc(request)
	if msgErr != nil {
		LogError("roomSettleLogic HundredGameSettleDiver settleFunc has err", msgErr)
		return request, msgErr
	}

	// 房间所有玩家游戏数+1
	for v, k := range request.PlayerInfo {
		if k.Uuid == "" {
			continue
		}
		request.PlayerInfo[v].PlayNum += 1
	}

	request.DoTime = nowTime + int64(settleTime)
	request.NextRoomState = pb.RoomState_RoomStateReady
	return request, nil
}
