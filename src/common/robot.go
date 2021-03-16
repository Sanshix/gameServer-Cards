package common

import (
	pb "gameServer-demo/src/grpc"

	"github.com/golang/protobuf/proto"
)

// GetRobotActionGroup 找到一个机器人所在的行为组
// 返回行为组的uuid，如果在空闲组，则返回"free"
// 如果不存在于任何组，则返回空字串
func GetRobotActionGroup(robotUUID string, robotManagerInfo *pb.RobotManagerInfo) string {
	// 先查看是不是在free组中
	freeIndex := IndexOf(robotManagerInfo.GetFreeRobot(), robotUUID)
	if freeIndex != -1 {
		return "free"
	}
	// 然后依次查找每个行为组
	for _, oneGroup := range robotManagerInfo.GetActionRobot() {
		actionIndex := IndexOf(oneGroup.GetRobotUuid(), robotUUID)
		if actionIndex != -1 {
			return oneGroup.GetActionGroupUuid()
		}
	}
	return ""
}

// GetRobotManagerInfo 获得机器人管理信息，不加锁
func GetRobotManagerInfo() *pb.RobotManagerInfo {
	extroInfo := &pb.MessageExtroInfo{}
	redisGetRequest := &pb.RedisMessage{}
	redisGetRequest.Table = RedisRobotManagerInfoTable
	redisGetReply := &pb.RedisMessage{}
	robotManagerInfo := &pb.RobotManagerInfo{}
	msgErr := Router.Call("Redis", "GetByte", redisGetRequest, redisGetReply, extroInfo)
	if msgErr != nil {
		LogError("GetRobotManagerInfo Call Redis Get has err", msgErr)
		return robotManagerInfo
	}

	robotManagerInfoByte := redisGetReply.ValueByte
	if robotManagerInfoByte != nil {
		err := proto.Unmarshal(robotManagerInfoByte, robotManagerInfo)
		if err != nil {
			LogError("GetRobotManagerInfo RobotManagerInfo proto.Unmarshal has err", err)
			return robotManagerInfo
		}
	}
	return robotManagerInfo
}

// HandleRobotManagerInfoFunc 处理机器人管理信息的回掉
type HandleRobotManagerInfoFunc func(*pb.RobotManagerInfo) *pb.ErrorMessage

// ChangeRobotManagerInfo 改变机器人管理信息，内部已加锁
func ChangeRobotManagerInfo(callBack HandleRobotManagerInfoFunc) *pb.ErrorMessage {
	extroInfo := &pb.MessageExtroInfo{}
	// 加锁操作
	infoMutex, err := Locker.MessageLock(MessageLockRobotManagerInfo, extroInfo, "ManagerInfo")
	if err != nil {
		LogError("ChangeRobotManagerInfo MessageLockRobotManagerInfo has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockRobotManagerInfo, extroInfo, "ManagerInfo", infoMutex)
	//先看redis中有没有
	redisGetRequest := &pb.RedisMessage{}
	redisGetRequest.Table = RedisRobotManagerInfoTable
	redisGetReply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "GetByte", redisGetRequest, redisGetReply, extroInfo)
	if msgErr != nil {
		LogError("ChangeRobotManagerInfo Call Redis Get has err", msgErr)
		return msgErr
	}
	robotManagerInfo := &pb.RobotManagerInfo{}
	robotManagerInfoByte := redisGetReply.ValueByte
	if robotManagerInfoByte != nil {
		err = proto.Unmarshal(robotManagerInfoByte, robotManagerInfo)
		if err != nil {
			LogError("ChangeRobotManagerInfo RobotManagerInfo proto.Unmarshal has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
	}
	msgErr = callBack(robotManagerInfo)
	if msgErr != nil {
		return msgErr
	}
	robotManagerInfoByte, err = proto.Marshal(robotManagerInfo)
	if err != nil {
		LogError("ChangeRobotManagerInfo RobotManagerInfo proto.Marshal has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	redisSetRequest := &pb.RedisMessage{}
	redisSetRequest.Table = RedisRobotManagerInfoTable
	redisSetRequest.ValueByte = robotManagerInfoByte
	redisSetReply := &pb.RedisMessage{}
	msgErr = Router.Call("Redis", "SetByte", redisSetRequest, redisSetReply, extroInfo)
	if msgErr != nil {
		LogError("ChangeRobotManagerInfo Call Redis Set has err", msgErr)
		return msgErr
	}
	return nil
}
