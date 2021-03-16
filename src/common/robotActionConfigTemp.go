package common

import (
	"errors"
	pb "gameServer-demo/src/grpc"
)

// RobotActionConfigTemp 机器人行为模版配置，提供默认模版
var RobotActionConfigTemp map[string]*pb.RobotActionConfig

func init() {
	RobotActionConfigTemp = make(map[string]*pb.RobotActionConfig)
	RobotActionConfigTemp["default-online"] = &pb.RobotActionConfig{
		ActionUuid:    "default-online",
		ActionName:    "默认上线行为",
		ActionType:    pb.RobotAction_RobotAction_Online,
		LoginInterval: 10,
	}
	RobotActionConfigTemp["default-offline"] = &pb.RobotActionConfig{
		ActionUuid: "default-offline",
		ActionName: "默认离线行为",
		ActionType: pb.RobotAction_RobotAction_Offline,
	}
	RobotActionConfigTemp["default-recharge"] = &pb.RobotActionConfig{
		ActionUuid:     "default-recharge",
		ActionName:     "默认充值行为",
		ActionType:     pb.RobotAction_RobotAction_Recharge,
		MinNum:         30000,
		MinRechargeNum: 10000000,
		MaxRechargeNum: 20000000,
	}

	// 推筒子
	RobotActionConfigTemp["default-pushbobbin-joinRoom"] = &pb.RobotActionConfig{
		ActionUuid:           "default-pushbobbin-joinRoom",
		ActionName:           "默认推筒子加入房间",
		ActionType:           pb.RobotAction_RobotAction_PushBobbin_JoinRoom,
		JoinRoomScenes:       []int32{1},
		JoinRoomScenesWeight: []int32{100},
		JoinRoomRobotLimit:   20,
	}
	RobotActionConfigTemp["default-pushbobbin-exitRoom"] = &pb.RobotActionConfig{
		ActionUuid: "default-pushbobbin-exitRoom",
		ActionName: "默认推筒子退出房间",
		ActionType: pb.RobotAction_RobotAction_PushBobbin_ExitRoom,
	}
	RobotActionConfigTemp["default-pushbobbin-play"] = &pb.RobotActionConfig{
		ActionUuid:               "default-pushbobbin-play",
		ActionName:               "默认推筒子玩耍",
		ActionType:               pb.RobotAction_RobotAction_PushBobbin_Play,
		MinPlayNum:               10,
		MaxPlayNum:               150,
		PlayEndPre:               10,
		MinBalance:               20000,
		RepeatBet:                30,
		PushBobbinBets:           []int32{30, 30, 30},
		PushBobbinBetMoneyWeight: []int32{60, 30, 10, 0, 0, 0},
	}
	RobotActionConfigTemp["default-pushbobbinBank-play"] = &pb.RobotActionConfig{
		ActionUuid:         "default-pushbobbinBank-play",
		ActionName:         "默认推筒子庄家玩耍",
		ActionType:         pb.RobotAction_RobotAction_PushBobbinBank_Play,
		MinPlayNum:         10,
		MaxPlayNum:         200,
		PlayEndPre:         1,
		MinBalance:         1000000,
		RobotDownBankRatio: 20,
		BanksLength:        5,
	}
}

// InitRobotActionConfigTemp 预设组件要用的机器人行为配置模版，如果变量已存在，则不重置
func InitRobotActionConfigTemp(initRobotActionConfigNameArr []string) error {
	for _, oneRobotActionConfigName := range initRobotActionConfigNameArr {
		oneConfig, ok := RobotActionConfigTemp[oneRobotActionConfigName]
		if !ok {
			return errors.New("common InitGlobleConfigTemp has err")
		}

		robotActionConfig := Configer.GetRobotActionConfig(oneRobotActionConfigName)
		if robotActionConfig == nil {
			Configer.AddRobotActionConfig(oneConfig)
		}
	}
	return nil
}
