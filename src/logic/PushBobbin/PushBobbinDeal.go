package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"
)

func init() {
	common.AllComponentMap["PushBobbinDeal"] = &PushBobbinDeal{}
}

// PushBobbinDeal 推筒子游戏组件，用于处理发牌阶段的逻辑
type PushBobbinDeal struct {
	base.Base
}

// LoadComponent 加载组件
func (obj *PushBobbinDeal) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *PushBobbinDeal) Start() {
	obj.Base.Start()

}

// Drive 房间发牌状态的驱动逻辑
func (obj *PushBobbinDeal) Drive(request *pb.RoomInfo, _ *pb.MessageExtroInfo) (*pb.RoomInfo, *pb.ErrorMessage) {
	//获取当前时间时间戳
	nowTime := time.Now().Unix()
	if request.NextRoomState != pb.RoomState_RoomStateDeal {
		if nowTime < request.DoTime {
			return request, nil
		}
		request.CurRoomState = pb.RoomState_RoomStateBet
		request.NextRoomState = pb.RoomState_RoomStateBet
		request.DoTime = nowTime
		return request, nil
	}
	dealTimeStr := common.GetRoomConfig(request, "DealTime")
	dealTime, err := strconv.Atoi(dealTimeStr)
	if err != nil {
		common.LogError("PushBobbinDeal Drive dealTimeStr has err", dealTimeStr)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	//推送消息
	roomState := &pb.PushRoomStateChange{
		RoomId:            request.GetUuid(),
		BeforeState:       pb.RoomState_RoomStateLocation,
		AfterState:        pb.RoomState_RoomStateDeal,
		AfterStateEndTime: nowTime + int64(dealTime),
	}
	PushRoomState := &pb.PushPushBobbinSendPoker{
		RoomStateChange: roomState,
		RoomId:          request.Uuid,
	}
	common.RoomBroadcast(request, PushRoomState)

	//下个状态
	request.NextRoomState = pb.RoomState_RoomStateBet
	request.DoTime = nowTime + int64(dealTime)
	return request, nil
}
