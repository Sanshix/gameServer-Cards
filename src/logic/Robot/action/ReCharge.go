package action

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
)

func init() {
}

// ReCharge 充值行为
type ReCharge struct {
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
func (o *ReCharge) Action(playerInfo *pb.PlayerInfo, roomInfo *pb.RoomInfo, actionConfig *pb.RobotActionConfig, extraInfo *pb.MessageExtroInfo) (bool, bool, int64) {
	// 如果大于了最低+最大充值金额，则要把钱扣下来
	if playerInfo.GetBalance() >= (actionConfig.GetMinNum() + actionConfig.GetMaxRechargeNum()) {
		realRechareNum := common.GetRandomNum(int(actionConfig.GetMinRechargeNum()), int(actionConfig.GetMaxRechargeNum()))
		playerInfo.Balance = actionConfig.GetMinNum() + int64(realRechareNum)

		savePlayerRequest := &pb.SavePlayerRequest{}
		savePlayerRequest.PlayerInfo = playerInfo
		savePlayerRequest.ForceSave = true
		msgErr := common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extraInfo)
		if msgErr != nil {
			common.LogError("ReCharge Action1 SavePlayer has err", msgErr)
			return false, true, 5
		}
		common.LogDebug("robot rechare1 ok", playerInfo.GetUuid())
		return true, false, 1
	}
	// 小于最大充值，走正常充值流程
	if playerInfo.GetBalance() >= actionConfig.GetMinNum() {
		return true, false, 1
	}
	realRechareNum := common.GetRandomNum(int(actionConfig.GetMinRechargeNum()), int(actionConfig.GetMaxRechargeNum()))
	playerInfo.Balance = playerInfo.GetBalance() + int64(realRechareNum)

	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = true
	msgErr := common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extraInfo)
	if msgErr != nil {
		common.LogError("ReCharge Action SavePlayer has err", msgErr)
		return false, true, 5
	}
	common.LogDebug("robot rechare ok", playerInfo.GetUuid())
	return true, false, 1
}
