package action

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"

	"github.com/golang/protobuf/ptypes"
)

func init() {
}

// PushBobbinJoinRoom 推筒子机器人进入房间行为
type PushBobbinJoinRoom struct {
}

// Action 触发行为
//
// 传入参数是机器人信息playerInfo可修改，且修改的信息在外部会保存
// roomInfo都是只读的，修改是无用的
// playerInfo也是只读的，修改需要自己保存一次，底层已对player加锁
//
// 返回值依次是：
//
// 行为是否结束，这里的行为结束意味着整个完成了，可以进行下一个行为了，而不是单次行为有没有完成
// 比如下注在玩耍行为下会有多次，而不是一次，如果传结束了，则整个玩耍行为会结束
//
// 行为是否出错，同一行为出错的话外部会进行累积，如果累积到一定数字，则认为机器人卡死，会在底层作出处理
//
// 下次行为在多少秒之后，用于顶层tick
func (o *PushBobbinJoinRoom) Action(playerInfo *pb.PlayerInfo, roomInfo *pb.RoomInfo, actionConfig *pb.RobotActionConfig, extraInfo *pb.MessageExtroInfo) (bool, bool, int64) {

	// 排除设置错误
	if roomInfo != nil {
		return true, false, 1
	}
	if playerInfo.IsRobot == false || playerInfo.Role != pb.Roles_Robot {
		common.LogError("机器人异常！", playerInfo)
		return true, false, 1
	}
	// 如果被标记为下岗，则结束行为
	if playerInfo.GetRobotExtroInfo().GetIsLaidOff() == true {
		return true, false, 1
	}
	if len(actionConfig.GetJoinRoomScenesWeight()) != len(actionConfig.GetJoinRoomScenes()) {
		common.LogError("PushBobbinJoinRoom Action scenes config and weight config err")
		return false, true, 5
	}
	if len(actionConfig.GetJoinRoomScenes()) <= 0 {
		common.LogError("PushBobbinJoinRoom Action scenes config err")
		return false, true, 5
	}

	// 通过权重比例随机选择机器人进入场次
	sceneIndex, err := common.GetRandomIndexByWeight(actionConfig.GetJoinRoomScenesWeight())
	if err != nil {
		common.LogError("PushBobbinJoinRoom Action get scene index err", err)
		return false, true, 5
	}

	//封禁 推筒子 加入房间的协议
	gameJoinRequest := &pb.GameJoinRoomRequest{}
	gameJoinRequest.GameScene = actionConfig.GetJoinRoomScenes()[sceneIndex]
	gameJoinRequest.JoinRoomRobotLimit = actionConfig.GetJoinRoomRobotLimit()
	gameJoinReply := &pb.GameJoinRoomReply{}

	PushBobbinDoContent, err := ptypes.MarshalAny(gameJoinRequest)
	if err != nil {
		common.LogError("PushBobbinJoinRoom Action MarshalAny err", err)
		return false, true, 5
	}
	PushBobbinDoRequest := &pb.PushBobbinDoRequest{}
	PushBobbinDoRequest.DoType = pb.PushBobbinDoType_PushBobbin_JoinRoom
	PushBobbinDoRequest.DoMessageContent = PushBobbinDoContent
	msgErr := common.Router.Call("PushBobbinRoute", "Do", PushBobbinDoRequest, gameJoinReply, extraInfo)
	if msgErr != nil {
		common.LogError("PushBobbinJoinRoom Action call do err", msgErr)
		return false, true, 5
	}
	common.LogDebug("robot PushBobbin joinRoom ok", playerInfo.GetUuid())
	return true, false, 1
}
