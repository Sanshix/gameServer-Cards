package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"
)

func init() {
	common.AllComponentMap["PushBobbinLocation"] = &PushBobbinLocation{}
}

// PushBobbinLocation 推筒子游戏的房间状态组件，用于处理定位阶段的逻辑(回合的第一个阶段）
type PushBobbinLocation struct {
	base.Base
}

// LoadComponent 加载组件
func (obj *PushBobbinLocation) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *PushBobbinLocation) Start() {
	obj.Base.Start()

}

// 定位阶段的主驱动
func (obj *PushBobbinLocation) Drive(request *pb.RoomInfo, _ *pb.MessageExtroInfo) (*pb.RoomInfo, *pb.ErrorMessage) {
	nowTime := time.Now().Unix()
	if request.NextRoomState != pb.RoomState_RoomStateLocation {
		if nowTime < request.DoTime {
			return request, nil
		}
		request.CurRoomState = pb.RoomState_RoomStateDeal
		request.NextRoomState = pb.RoomState_RoomStateDeal
		request.DoTime = nowTime
		return request, nil
	}
	//玩家当庄最低金额
	MinMoneyStr := common.GetRoomConfig(request, "UserBankerMoney")
	MinMoney, err := strconv.Atoi(MinMoneyStr)
	if err != nil {
		common.LogError("PushBobbinFree Diver MinMoneyStr has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//玩家当庄最大回合数
	MaxRoundStr := common.GetRoomConfig(request, "UserBankerRound")
	MaxRound, err := strconv.Atoi(MaxRoundStr)
	if err != nil {
		common.LogError("PushBobbinFree Diver MaxRoundStr has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//系统当庄默认金额
	DefaultMoneyStr := common.GetRoomConfig(request, "DefaultBankerMoney")
	DefaultMoney, err := strconv.Atoi(DefaultMoneyStr)
	if err != nil {
		common.LogError("PushBobbinFree Diver DefaultMoneyStr has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 4.检测更换庄家
	pushBanker := &pb.PushPushBobbinBankerMessage{
		RoomId:              request.GetUuid(),
		BeforeBanker:        request.GetBankerUuid(),
		ReplaceBankerReason: pb.KickBankerReason_KickBankerNone,
	}

	// 更新庄家坐庄次数
	request.BankerNowRound++
	var tempBankers []string
	// 检测庄家队列中金币小于上庄最低金额的玩家
	for k, v := range request.Bankers {
		if !common.PlayerMoneyEnoughOrInRoom(v, int64(MinMoney), request) {
			//更新移除队列中的玩家的状态为空闲
			tempPlayer := common.GetRoomPlayerInfo(request, v)
			if tempPlayer != nil {
				tempPlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateFree
			}
		} else {
			// 更新庄家队列
			tempBankers = append(tempBankers, request.Bankers[k])
		}
	}

	request.Bankers = tempBankers
	// 4.1当房间庄家是玩家时
	if request.GetBankerUuid() != "" && request.GetBankerUuid() != "systemBanker" {
		// 容错庄家离线被kick
		pushBanker.ReplaceBankerReason = pb.KickBankerReason_KickBankerByRoom
		for _, k := range request.GetPlayerInfo() {
			if k.Uuid == request.BankerUuid {
				pushBanker.ReplaceBankerReason = pb.KickBankerReason_KickBankerNone
				// 钱不够就赋值庄家改变原因 是 钱不够
				if k.GetBalance() < int64(MinMoney) && request.DownBankerQuest == false {
					pushBanker.ReplaceBankerReason = pb.KickBankerReason_KickBankerByMoney
				}
				break
			}
		}
		//坐庄回合达到最高回合次数
		if request.GetBankerNowRound() >= int64(MaxRound) {
			pushBanker.ReplaceBankerReason = pb.KickBankerReason_KickBankerByRound
		} else if request.DownBankerQuest == true {
			// 庄家主动申请下庄
			pushBanker.ReplaceBankerReason = pb.KickBankerReason_KickBankerBySelf
		} else if pushBanker.ReplaceBankerReason != pb.KickBankerReason_KickBankerByMoney &&
			pushBanker.ReplaceBankerReason != pb.KickBankerReason_KickBankerByRoom {
			pushBanker.ReplaceBankerReason = pb.KickBankerReason_KickBankerNone
		}
	}
	// 4.2 根据庄家是否需要改变 进行充填
	// 庄家为系统或者空时 也是为 庄家需要改变
	// 当改变时,先将庄家改变为默认系统
	// 再根据申请庄家队列是否有人来取人
	if pushBanker.ReplaceBankerReason != pb.KickBankerReason_KickBankerNone || request.BankerUuid == "" || request.BankerUuid == "systemBanker" {
		if pushBanker.ReplaceBankerReason != pb.KickBankerReason_KickBankerNone && request.BankerUuid != "" {
			tempInfo := common.GetRoomPlayerInfo(request, request.BankerUuid)
			if tempInfo != nil {
				tempInfo.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateFree
			}
		}
		request.BankerNowRound = 0
		request.BankerUuid = "systemBanker"
		request.DownBankerQuest = false
		// 当庄家申请队列里面有人时，取队列第一个人，并将其从申请庄家队列删除
		if len(request.Bankers) >= 1 {
			request.BankerUuid = request.Bankers[0]
			request.Bankers = request.Bankers[1:]
		}
	}
	// 这个时候庄家还是"" 就说明有误
	if request.BankerUuid == "" {
		common.LogError("PushBobbinLocation Drive has error: BankerUuid == empty")
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 4.3 限红判断（直接给总数让客户端判断）
	pushBanker.MaxBetRatio = make([]int64, 1)
	// 系统
	if request.BankerUuid == "systemBanker" {
		pushBanker.MaxBetRatio[0] = int64(DefaultMoney)
		request.MaxBetRatio = pushBanker.MaxBetRatio
		//玩家
	} else {
		for _, k := range request.PlayerInfo {
			if k.Uuid == request.BankerUuid {
				pushBanker.MaxBetRatio[0] = k.Balance
				request.MaxBetRatio = pushBanker.MaxBetRatio
				break
			}
		}
	}

	// 4.4 庄家信息推送
	pushBanker.Bankers = request.Bankers
	pushBanker.AfterBanker = request.BankerUuid
	pushBanker.NowRound = request.BankerNowRound
	common.RoomBroadcast(request, pushBanker)
	common.LogDebug("庄家列表：", pushBanker)

	// 新回合准备设置-------------------end----------------------

	//下个状态
	request.NextRoomState = pb.RoomState_RoomStateDeal

	return request, nil
}
