package action

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"github.com/golang/protobuf/ptypes"
)

func init() {
}

// PushBobbinPlay 推筒子机器人玩耍行为
type PushBobbinPlay struct {
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
func (o *PushBobbinPlay) Action(playerInfo *pb.PlayerInfo, roomInfo *pb.RoomInfo, actionConfig *pb.RobotActionConfig, extraInfo *pb.MessageExtroInfo) (bool, bool, int64) {
	// 当玩家房间信息为空时，结束当前行为
	if roomInfo == nil {
		return true, false, 5
	}

	roomPlayerInfo := common.GetRoomPlayerInfo(roomInfo, playerInfo.GetUuid())
	if roomPlayerInfo == nil {
		common.LogError("PushBobbinPlay Action player not in room", playerInfo.GetUuid())
		return true, false, 5
	}

	// 当机器人没得什么钱了，就随缘观战一会退出去充钱
	if roomPlayerInfo.PlayerRoomState != pb.PlayerRoomState_PlayerRoomStatePlay && roomPlayerInfo.Balance < actionConfig.MinBalance {
		return true, false, int64(common.GetRandomNum(3, 20))
	}

	// 如果当前房间是下注状态
	if roomInfo.GetCurRoomState() == pb.RoomState_RoomStateBet {

		// 如果被标记为下岗，则结束行为
		if playerInfo.GetRobotExtroInfo().GetIsLaidOff() == true {
			return true, false, 1
		}

		// 如果大于了最大玩耍局数，则结束行为
		if roomPlayerInfo.GetPlayNum() >= actionConfig.GetMaxPlayNum() {
			return true, false, 1
		}

		// 如果介于最大和最小局数之间，则随机结束行为
		if roomPlayerInfo.GetPlayNum() >= actionConfig.GetMinPlayNum() &&
			roomPlayerInfo.GetPlayNum() < actionConfig.GetMaxPlayNum() {
			randomNum := common.GetRandomNum(1, 100)
			if int32(randomNum) <= actionConfig.GetPlayEndPre() {
				return true, false, 1
			}
		}

		// 随缘延迟
		if int64(common.GetRandomNum(1, 3)) == 1 {
			return false, false, 1
		}

		// 获取房间下的总注
		roomAllBet := roomInfo.GetAllBet()
		if len(roomAllBet) != 4 { //是房间的错误，
			common.LogError("PushBobbinPlay Action GetPushBobbinAllBet has err: allBet length != 3")
			return false, true, 5
		}

		// 随缘投注 根据房间总注反向随缘
		var betMoney int64
		var betArea pb.PushBobbinBetArea

		BetIndex, err := common.GetRandomIndexByWeight(actionConfig.PushBobbinBetMoneyWeight)
		for err != nil {
			common.LogError("PushBobbinPlay Action get scene index err", err)
			return false, true, 1
		}
		// 获取下注区域和下注金额
		betMoney = getBetMoney(roomInfo, BetIndex)
		betAreaInt := getArea(actionConfig.PushBobbinBets)
		betArea = pb.PushBobbinBetArea(betAreaInt)
		if betMoney > roomInfo.MaxBetRatio[0] {
			// 下注金额超出限红
			return false, false, 4
		}
		// 当投注金额为0时，随缘重新加载
		if betMoney == 0 {
			return false, false, int64(common.GetRandomNum(1, 2))
		}

		// 当机器人金额小于投注金额,随缘重新加载
		if roomPlayerInfo.Balance < betMoney {
			return false, false, int64(common.GetRandomNum(1, 2))
		}

		// 当机器人在这局已经下过注了，按概率判断是否继续下注
		if roomPlayerInfo.PlayNum == actionConfig.LastBetNum && actionConfig.LastBetNum != 0 {
			if common.GetRandomNum(1, 100) > int(actionConfig.RepeatBet) {
				return false, false, 3
			}
		}

		// 投注 操作封装
		gameBetRequest := &pb.PushBobbinBetRequest{
			BetArea:    betArea,
			BetBalance: betMoney,
		}
		PushBobbinDoContent, err := ptypes.MarshalAny(gameBetRequest)
		if err != nil {
			common.LogError("PushBobbinPlay Action gameBetRequest MarshalAny err", err)
			return false, true, 5
		}
		request := &pb.PushBobbinDoRequest{
			DoType:           pb.PushBobbinDoType_PushBobbin_PlayerBet,
			DoMessageContent: PushBobbinDoContent,
		}
		reply := &pb.PushBobbinBetReply{}
		msgErr := common.Router.Call("PushBobbinRoute", "Do", request, reply, extraInfo)
		if msgErr != nil {
			common.LogError("PushBobbinPlay Action gameBetRequest call do err", msgErr)
			return false, true, 5
		}
		common.LogDebug("推筒子下注成功，", gameBetRequest)
		// 赋值给机器人当前下注局数
		actionConfig.LastBetNum = roomPlayerInfo.PlayNum
		// 随缘加载
		return false, false, int64(common.GetRandomNum(1, 4))
	}
	// 随缘加载
	return false, false, int64(common.GetRandomNum(2, 3))
}
