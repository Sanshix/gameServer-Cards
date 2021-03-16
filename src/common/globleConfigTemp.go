package common

import (
	"errors"
	pb "gameServer-demo/src/grpc"
)

// GlobleConfigTemp 这里定义了全局配置的默认配置模板，外部需要自己初始化需要的配置，争取架构在空表的情况下可以完成初始化
var GlobleConfigTemp map[string]*pb.GlobalConfig

func init() {
	GlobleConfigTemp = make(map[string]*pb.GlobalConfig)
	GlobleConfigTemp["PushBobbinServerNum"] = &pb.GlobalConfig{
		Name:   "PushBobbinServerNum",
		Value:  "1",
		Remark: "推筒子的房间线路数量,新增线路必须先创建服务后再配置",
	}
	GlobleConfigTemp["PushBobbinMaxRoomNumOneServer"] = &pb.GlobalConfig{
		Name:   "PushBobbinMaxRoomNumOneServer",
		Value:  "100",
		Remark: "推筒子在一个线路上的房间最大数量",
	}
	GlobleConfigTemp["GetRecordDay"] = &pb.GlobalConfig{
		Name:   "GetRecordDay",
		Value:  "7",
		Remark: "获取最近多少天的战绩记录",
	}
	GlobleConfigTemp["SocketServerNum"] = &pb.GlobalConfig{
		Name:   "SocketServerNum",
		Value:  "1",
		Remark: "socket服务器的数量",
	}
	GlobleConfigTemp["OpenPlayerStartPlayGameNotice"] = &pb.GlobalConfig{
		Name:   "OpenPlayerStartPlayGameNotice",
		Value:  "true",
		Remark: "是否开启玩家开始游戏通知",
	}
}

// InitGlobleConfigTemp 预设组件要用的全局配置模版，如果变量已存在，则不重置
func InitGlobleConfigTemp(initGlobleConfigNameArr []string) error {
	for _, oneGlobleConfigName := range initGlobleConfigNameArr {
		oneConfig, ok := GlobleConfigTemp[oneGlobleConfigName]
		if !ok {
			return errors.New("common InitGlobleConfigTemp has err")
		}
		Configer.SetGlobal(oneConfig, false)
	}
	return nil
}
