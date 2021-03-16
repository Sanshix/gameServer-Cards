package action

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"

	"github.com/golang/protobuf/ptypes"
)

func init() {
}

// PushBobbinExitRoom 推筒子机器人退出房间行为
type PushBobbinExitRoom struct {
}

// Action 触发行为
//
// 传入参数是机器人信息playerInfo可修改，且修改的信息在外部会保存
// roomInfo都是只读的，修改是无用的
// playerInfo也是只读的，修改需要自己保存一次，底层已对player加锁
//
// 返回值依次是：
//push_bobbin_driver
// 行为是否结束，这里的行为结束意味着整个完成了，可以进行下一个行为了，而不是单次行为有没有完成
// 比如下注在玩耍行为下会有多次，而不是一次，如果传结束了，则整个玩耍行为会结束
//
// 行为是否出错，同一行为出错的话外部会进行累积，如果累积到一定数字，则认为机器人卡死，会在底层作出处理
//
// 下次行为在多少秒之后，用于顶层tick
func (o *PushBobbinExitRoom) Action(playerInfo *pb.PlayerInfo, roomInfo *pb.RoomInfo, actionConfig *pb.RobotActionConfig, extraInfo *pb.MessageExtroInfo) (bool, bool, int64) {
	if roomInfo == nil {
		return true, false, 1
	}
	// 如果被标记为下岗，则结束行为
	if playerInfo.GetRobotExtroInfo().GetIsLaidOff() == true {
		return true, false, 1
	}

	//获取玩家在房间的索引
	var playerIndex = -1
	for v, k := range roomInfo.PlayerInfo {
		if k.GetUuid() == playerInfo.GetUuid() {
			playerIndex = v
			break
		}
	}
	if playerIndex == -1 { // 此处应该提交报错，出现这个错误有可能锁卡了?
		common.LogError("PushBobbinExitRoom Action playerIndex == -1,but roomInfo != nil!")
		return false, true, 1
	}

	// 如果玩家不在游戏状态即可退出
	if roomInfo.PlayerInfo[playerIndex].GetPlayerRoomState() != pb.PlayerRoomState_PlayerRoomStatePlay {
		gameExitRoomRequest := &pb.GameExitRoomRequest{}
		gameExitRoomReply := &pb.GameExitRoomReply{}

		PushBobbinDoContent, err := ptypes.MarshalAny(gameExitRoomRequest)
		if err != nil {
			common.LogError("PushBobbinExitRoom Action MarshalAny err", err)
			return false, true, 5
		}
		PushBobbinDoRequest := &pb.PushBobbinDoRequest{}
		PushBobbinDoRequest.DoType = pb.PushBobbinDoType_PushBobbin_ExitRoom
		PushBobbinDoRequest.DoMessageContent = PushBobbinDoContent
		msgErr := common.Router.Call("PushBobbinRoute", "Do", PushBobbinDoRequest, gameExitRoomReply, extraInfo)
		if msgErr != nil {
			common.LogError("PushBobbinExitRoom Action call do err", msgErr)
			return false, true, 5
		}
		common.LogDebug("robot PushBobbin ExitRoom  ok", playerInfo.GetUuid())
		return true, false, 1
	}
	return false, false, 5
}
