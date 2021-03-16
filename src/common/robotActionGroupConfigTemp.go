package common

import (
	"errors"
	pb "gameServer-demo/src/grpc"
)

// RobotActionGroupConfigTemp 机器人行为模版配置，提供默认模版
var RobotActionGroupConfigTemp map[string]*pb.RobotActionGroupConfig

func init() {
	RobotActionGroupConfigTemp = make(map[string]*pb.RobotActionGroupConfig)
	// 推筒子
	RobotActionGroupConfigTemp["default-push-bobbin-robot"] = &pb.RobotActionGroupConfig{
		ActionGroupUuid: "default-push-bobbin-robot",
		ActionGroupName: "默认推筒子机器人",
		ActionConfigsUuid: []string{
			"default-online",
			"default-recharge",
			"default-pushbobbin-joinRoom",
			"default-pushbobbin-play",
			"default-pushbobbin-exitRoom",
			"default-offline",
		},
		RobotNum: 2,
	}

	// 推筒子庄家机器人
	RobotActionGroupConfigTemp["default-push-bobbin-bank-robot"] = &pb.RobotActionGroupConfig{
		ActionGroupUuid: "default-push-bobbin-bank-robot",
		ActionGroupName: "默认推筒子庄家机器人",
		ActionConfigsUuid: []string{
			"default-online",
			"default-recharge",
			"default-pushbobbin-joinRoom",
			"default-pushbobbinBank-play",
			"default-pushbobbin-exitRoom",
			"default-offline",
		},
		RobotNum: 2,
	}
}

// InitRobotActionGroupConfigTemp 预设组件要用的机器人行为配置模版，如果变量已存在，则不重置
func InitRobotActionGroupConfigTemp(initRobotActionGroupConfigNameArr []string) error {
	for _, oneRobotActionGroupConfigName := range initRobotActionGroupConfigNameArr {
		oneConfig, ok := RobotActionGroupConfigTemp[oneRobotActionGroupConfigName]
		if !ok {
			return errors.New("common InitGlobleConfigTemp has err")
		}
		robotActionGroupConfig := Configer.GetRobotActionGroupConfig(oneRobotActionGroupConfigName)
		if robotActionGroupConfig == nil {
			Configer.AddRobotActionGroupConfig(oneConfig)
		}
	}
	return nil
}
