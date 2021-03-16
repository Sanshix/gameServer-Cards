package action

import (
	pb "gameServer-demo/src/grpc"
)

func init() {
}

// BaseActionI 行为接口定义
type BaseActionI interface {
	Action(playerInfo *pb.PlayerInfo, roomInfo *pb.RoomInfo, actionConfig *pb.RobotActionConfig, extraInfo *pb.MessageExtroInfo) (bool, bool, int64)
}
