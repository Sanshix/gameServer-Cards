package action

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"github.com/golang/protobuf/ptypes"
	"strconv"
)

func init() {
}

// PushBobbinBankPlay 推筒子庄家机器人玩耍行为
type PushBobbinBankPlay struct {
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
func (o *PushBobbinBankPlay) Action(playerInfo *pb.PlayerInfo, roomInfo *pb.RoomInfo, actionConfig *pb.RobotActionConfig, extraInfo *pb.MessageExtroInfo) (bool, bool, int64) {
	// 当玩家房间信息为空时，结束当前行为
	if roomInfo == nil {
		return true, false, 5
	}
	//common.LogDebug("PushBobbinBankRobot is Playing!!!")

	roomPlayerInfo := common.GetRoomPlayerInfo(roomInfo, playerInfo.GetUuid())
	if roomPlayerInfo == nil {
		common.LogError("PushBobbinRankPlay Action player not in room", playerInfo.GetUuid())
		return true, false, 5
	}

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

	userBankerMoneyStr := common.GetRoomConfig(roomInfo, "UserBankerMoney")
	userBankerMoney, err := strconv.Atoi(userBankerMoneyStr)
	if err != nil {
		common.LogError("PushBobbinRankPlay RequestUpBanker userBankerMoney has err", err)
		return false, true, 1
	}

	// 机器人是庄家，每次都有 X %几率下庄
	if roomPlayerInfo.Uuid == roomInfo.BankerUuid && common.GetRandomNum(1, 100) <= int(actionConfig.RobotDownBankRatio) {
		common.LogDebug("PushBobbin Banker DownBankRequest!")
		// 下庄 操作封装
		DownBankRequest := &pb.PushBobbinDownBankerRequest{}
		PushBobbinDoContent, err := ptypes.MarshalAny(DownBankRequest)
		if err != nil {
			common.LogError("PushBobbinRankPlay Action DownBankRequest MarshalAny err", err)
			return false, true, 5
		}
		request := &pb.PushBobbinDoRequest{
			DoType:           pb.PushBobbinDoType_PushBobbin_DownBanker,
			DoMessageContent: PushBobbinDoContent,
		}
		reply := &pb.PushBobbinDownBankerReply{}
		msgErr := common.Router.Call("PushBobbinRoute", "Do", request, reply, extraInfo)
		if msgErr != nil {
			common.LogError("PushBobbinRankPlay Action DownBankRequest call do err", msgErr)
			return false, true, 5
		}
		return false, false, 30
	}

	// 当庄家钱不够上庄时并且不是玩耍准备时,观战30s后离场去充钱
	if roomPlayerInfo.Balance < int64(userBankerMoney) && roomPlayerInfo.PlayerRoomState != pb.PlayerRoomState_PlayerRoomStatePlay {
		return true, false, 30
	}

	// 机器人不是庄家，并且钱够，庄家列表有空位时可以申请上庄
	if !alreadyInRank(playerInfo.Uuid, roomInfo) && len(roomInfo.Bankers) < int(actionConfig.BanksLength) && roomPlayerInfo.Balance >= int64(userBankerMoney) {
		// 上庄 操作封装1
		UpBankRequest := &pb.PushBobbinUpBankerRequest{}
		PushBobbinDoContent, err := ptypes.MarshalAny(UpBankRequest)
		if err != nil {
			common.LogError("PushBobbinBankPlay Action UpBankRequest MarshalAny err", err)
			return false, true, 5
		}
		request := &pb.PushBobbinDoRequest{
			DoType:           pb.PushBobbinDoType_PushBobbin_UpBanker,
			DoMessageContent: PushBobbinDoContent,
		}
		reply := &pb.PushBobbinUpBankerReply{}
		msgErr := common.Router.Call("PushBobbinRoute", "Do", request, reply, extraInfo)
		if msgErr != nil {
			common.LogError("PushBobbinBankPlay Action UpBankRequest call do err", msgErr)
			return false, true, 5
		}
		return false, false, 30
	}
	// 该机器人每60s才操作一次上下庄行为
	return false, false, 30
}
