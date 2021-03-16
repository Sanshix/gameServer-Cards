package action

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"time"
)

func init() {
}

// OffLine 离线行为
type OffLine struct {
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
func (o *OffLine) Action(playerInfo *pb.PlayerInfo, roomInfo *pb.RoomInfo, actionConfig *pb.RobotActionConfig, extraInfo *pb.MessageExtroInfo) (bool, bool, int64) {
	msgErr := common.Pusher.SetOffline(playerInfo.GetUuid())
	if msgErr != nil {
		common.LogError("OnLine Action SetOnline has err", msgErr)
		return false, true, 5
	}
	if playerInfo.GetRobotExtroInfo().GetIsLaidOff() == true {
		playerInfo.GetRobotExtroInfo().IsLaidOff = false
		playerInfo.GetRobotExtroInfo().ActionGroupUuid = ""
		playerInfo.GetRobotExtroInfo().CurActionUuid = ""
		playerInfo.GetRobotExtroInfo().LastOffLineTime = time.Now().Unix()
	}

	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = false
	msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extraInfo)
	if msgErr != nil {
		common.LogError("OffLine Action SavePlayer has err", msgErr)
		return false, true, 5
	}
	common.LogDebug("robot offline ok", playerInfo.GetUuid())
	return true, false, 1
}
