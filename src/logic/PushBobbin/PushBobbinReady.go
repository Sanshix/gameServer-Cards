package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"
)

func init() {
	common.AllComponentMap["PushBobbinReady"] = &PushBobbinReady{}
}

// PushBobbinReady 推筒子游戏的准备组件，用于处理准备阶段的逻辑
type PushBobbinReady struct {
	base.Base
}

// LoadComponent 加载组件
func (obj *PushBobbinReady) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *PushBobbinReady) Start() {
	obj.Base.Start()
	common.InitGameConfigTemp(common.PushBobbinGameConfigTemp, pb.GameType_PushBobbin)
}

// 推筒子准备阶段的主驱动
func (obj *PushBobbinReady) Drive(request *pb.RoomInfo, extraInfo *pb.MessageExtroInfo) (*pb.RoomInfo, *pb.ErrorMessage) {
	return common.HundredGameReadyDiver(request, func(roomInfo *pb.RoomInfo) *pb.ErrorMessage {

		readyTimeStr := common.GetRoomConfig(request, "ReadyTime")
		readyTime, err := strconv.Atoi(readyTimeStr)
		if err != nil {
			return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		// 定位-------------------------------start------------------
		// 1234 从庄家开始数 对应庄、天、顺、地
		tempDom := common.GetRandomNum(1, 6)
		tempDoms := common.GetRandomNum(1, 6)
		randomIndex := tempDom + tempDoms
		randomIndex %= 4
		if randomIndex == 0 {
			randomIndex = 4
		}
		areaIndex := pb.PushBobbinCardArea(randomIndex)

		roomState := &pb.PushRoomStateChange{
			RoomId:            request.GetUuid(),
			BeforeState:       pb.RoomState_RoomStateSettle,
			AfterState:        pb.RoomState_RoomStateReady,
			AfterStateEndTime: time.Now().Unix() + int64(readyTime),
		}
		PushRoomState := &pb.PushPushBobbinLocation{
			RoomStateChange: roomState,
			Index:           areaIndex,
			RoomId:          request.Uuid,
			Dice:            []int64{int64(tempDom), int64(tempDoms)},
		}
		common.RoomBroadcast(request, PushRoomState)
		// 定位---------------------------------end-------------------
		return nil
	})
}
