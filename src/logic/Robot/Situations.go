package logic

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"gameServer-demo/src/logic/Robot/action"
)

// ActionList 所有的行动列表
var ActionList map[pb.RobotAction]action.BaseActionI

func init() {
	ActionList = make(map[pb.RobotAction]action.BaseActionI)
	ActionList[pb.RobotAction_RobotAction_Offline] = &action.OffLine{}
	ActionList[pb.RobotAction_RobotAction_Online] = &action.OnLine{}
	ActionList[pb.RobotAction_RobotAction_Recharge] = &action.ReCharge{}
	// 推筒子
	ActionList[pb.RobotAction_RobotAction_PushBobbin_JoinRoom] = &action.PushBobbinJoinRoom{}
	ActionList[pb.RobotAction_RobotAction_PushBobbin_Play] = &action.PushBobbinPlay{}
	ActionList[pb.RobotAction_RobotAction_PushBobbinBank_Play] = &action.PushBobbinBankPlay{}
	ActionList[pb.RobotAction_RobotAction_PushBobbin_ExitRoom] = &action.PushBobbinExitRoom{}
}

// InitRobotConfigByOpenAction 通开放的行为初始化配置
func InitRobotConfigByOpenAction(oneOpenAction pb.RobotAction) {
	switch oneOpenAction {
	case pb.RobotAction_RobotAction_Online:
		_ = common.InitRobotActionConfigTemp([]string{"default-online"})
	case pb.RobotAction_RobotAction_Offline:
		_ = common.InitRobotActionConfigTemp([]string{"default-offline"})
	case pb.RobotAction_RobotAction_Recharge:
		_ = common.InitRobotActionConfigTemp([]string{"default-recharge"})
	// 推筒子
	case pb.RobotAction_RobotAction_PushBobbin_JoinRoom:
		_ = common.InitRobotActionConfigTemp([]string{
			"default-pushbobbin-joinRoom",
			"default-pushbobbin-exitRoom",
			"default-pushbobbin-play",
			"default-pushbobbinBank-play",
		})
		_ = common.InitRobotActionGroupConfigTemp([]string{"default-push-bobbin-robot"})
		_ = common.InitRobotActionGroupConfigTemp([]string{"default-push-bobbin-bank-robot"})
	}

}
