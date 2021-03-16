package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"math"
	"strconv"
	"strings"
	"time"
)

func init() {
	common.AllComponentMap["PushBobbinSettle"] = &PushBobbinSettle{}
}

// PushBobbinReady 推筒子游戏的结算组件，用于处理下注阶段的逻辑
type PushBobbinSettle struct {
	base.Base
}

// 最大赔率
var MaxOddsNumber int64

// LoadComponent 加载组件
func (obj *PushBobbinSettle) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *PushBobbinSettle) Start() {
	obj.Base.Start()
}

// 推筒子结算组件主驱动
func (obj *PushBobbinSettle) Drive(request *pb.RoomInfo, _ *pb.MessageExtroInfo) (*pb.RoomInfo, *pb.ErrorMessage) {
	return common.HundredGameSettleDiver(request, obj.realDrive)
}

// 推筒子结算组件主logic
func (obj *PushBobbinSettle) realDrive(request *pb.RoomInfo) *pb.ErrorMessage {
	// 获取当前时间时间戳
	nowTime := time.Now().Unix()

	// 获取结算时间
	settleTimeStr := common.GetRoomConfig(request, "SettleTime")
	settleTime, err := strconv.Atoi(settleTimeStr)
	if err != nil {
		common.LogError("PushBobbinSettle Drive settleTimeStr has err", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 获取抽数比例
	CommissionStr := common.GetRoomConfig(request, "Commission")
	Commission, err := strconv.ParseInt(CommissionStr, 10, 64)
	if err != nil {
		common.LogError("PushBobbinSettle Drive CommissionStr has err", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 获取最大赔率
	oddsStr := common.GetRoomConfig(request, "OddsDigit12")
	oddsNum, err := strconv.Atoi(oddsStr)
	MaxOddsNumber = int64(oddsNum)
	if err != nil {
		common.LogError("HundredBullSettle Drive Bull20Odds has err", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 输赢信息
	winInfos := pb.PushBobbinWinInfo{
		Time:      nowTime,
		RoomRound: request.RoomRound,
	}
	// 庄家是否是机器人
	bankerIsRobot := obj.isBankerIsRobot(request.PlayerInfo, request.BankerUuid)
	// 控制模式 0随机 1庄家赢 2控制闲家赢
	cheatType := 0
	// 获取血池状态
	bloodState := common.BloodGetState(request.GameType, request.GameScene)

	if bloodState == pb.BloodSlotStatus_BloodSlotStatus_Win {
		if bankerIsRobot {
			// 庄家是系统或者机器人的时候，控制庄家赢
			cheatType = 1
		} else {
			// 庄家是玩家的时候，控制闲家赢
			cheatType = 2
		}
	} else if bloodState == pb.BloodSlotStatus_BloodSlotStatus_Lose {
		if bankerIsRobot {
			// 闲家赢
			cheatType = 2
		} else {
			// 庄家赢
			cheatType = 1
		}
	}

	//获取牌堆
	cardHeap := getPushBobbinMahjongHeap()
	//切乱牌序
	common.RandSlice(cardHeap)
	// 大牌型
	maxCardType := 10
	// 小牌型
	minCardType := 0
	// 获取下注数最多的2个区域
	betX, betY := OrderBets(request.PlayerInfo, bankerIsRobot)
	if betX == 8 {
		// 没人下注 随机
		cheatType = 0
	}
	// 庄家牌型赔率
	var bankerOdds int64
	// common.LogDebug("血池状态:", bloodState, cheatType)
	// 给4个牌区开翻牌结果
	for key := 0; key < 4; key++ {
		// 默认随机开牌
		pokerList := cardHeap[:2]
		// 判断血池控制-------👇-------------------
		if key == 0 {
			// 庄家
			weightTemp := 0
			if cheatType == 1 {
				// 获取大牌
				pokerList, weightTemp, cardHeap = PushBobbinGetAssignMahjong(request, 1, maxCardType, cardHeap)
				maxCardType = weightTemp
			} else if cheatType == 2 {
				// 获取小牌
				pokerList, weightTemp, cardHeap = PushBobbinGetAssignMahjong(request, 2, 5, cardHeap)
				minCardType = weightTemp
			} else {
				// 按照随机发牌更新牌堆
				cardHeap = cardHeap[2:]
			}
		} else if key != betX && key != betY && cheatType != 0 {
			// 避免每个区域都强制输赢--只限制特殊牌型
			if cheatType == 1 {
				// 大概率赢
				pokerList, _, cardHeap = PushBobbinGetAssignMahjong(request, 4, 10, cardHeap)
			} else {
				// 大概率输
				pokerList, _, cardHeap = PushBobbinGetAssignMahjong(request, 4, 5, cardHeap)
			}
		} else {
			// 强控下注区域最多的2个区域
			// 闲家
			if cheatType == 1 {
				// 获取小牌
				pokerList, _, cardHeap = PushBobbinGetAssignMahjong(request, 2, maxCardType, cardHeap)
			} else if cheatType == 2 {
				// 获取大牌
				pokerList, _, cardHeap = PushBobbinGetAssignMahjong(request, 3, minCardType, cardHeap)
			} else {
				// 按照随机发牌更新牌堆
				cardHeap = cardHeap[2:]
			}

		}
		// ------------------👆-------------------

		// 存储开牌结果
		request.PushBobbinMahjongList = append(request.PushBobbinMahjongList, &pb.PushBobbinMahjong{
			MahjongList: pokerList,
		})
		// 获取牌型
		pokerType, pokerOdds, msgErr := getMahjongType(request, pokerList)
		if msgErr != nil {
			return msgErr
		}
		// 存储牌型
		request.PushBobbinMahjongList[key].MahjongType = pokerType
		//跳过庄家
		if key == 0 {
			bankerOdds = pokerOdds
			continue
		}
		//判断输赢 1为赢  2为输
		var winOrLose int64
		bankerType := request.PushBobbinMahjongList[0].MahjongType
		//牌型大小：五小牛>炸弹牛>同花牛>葫芦牛>顺子牛>五花牛>牛牛>有牛（从牛9到牛1）>无牛
		if bankerType > pokerType {
			//庄赢
			winOrLose = 2
		} else if bankerType < pokerType {
			//闲赢
			winOrLose = 1
		} else {
			//同牌型单独比较
			//获取庄家牌
			bankePokers := request.PushBobbinMahjongList[0].MahjongList
			// 都是0则庄家赢
			if pokerType == pb.PushBobbinMahjongType_PushBobbinMahjongType_None {
				winOrLose = 2
			} else {
				wins := obj.getCompareType(bankePokers, pokerList)
				if wins == true {
					winOrLose = 2
				} else {
					winOrLose = 1
				}
			}
		}
		winInfos.AllAreaOdds = append(winInfos.AllAreaOdds, pokerOdds)
		winInfos.AllAreaType = append(winInfos.AllAreaType, pokerType)
		winInfos.AllAreaWin = append(winInfos.AllAreaWin, winOrLose)
	}
	// 清理牌堆
	request.PokerCardHeap = []*pb.Poker{}
	// 保存输赢记录
	//common.LogDebug(winInfos)
	request.PushBobbinWinInfos = append([]*pb.PushBobbinWinInfo{&winInfos}, request.PushBobbinWinInfos...)
	if len(request.PushBobbinWinInfos) > 50 {
		request.PushBobbinWinInfos = request.PushBobbinWinInfos[:len(request.PushBobbinWinInfos)-1]
	}
	// 给房间所有人推送结算消息
	roomState := &pb.PushRoomStateChange{
		RoomId:            request.GetUuid(),
		BeforeState:       pb.RoomState_RoomStateBet,
		AfterState:        pb.RoomState_RoomStateSettle,
		AfterStateEndTime: nowTime + int64(settleTime),
	}
	PushRoomState := &pb.PushPushBobbinSettle{
		RoomStateChange: roomState,
		MahjongList:     request.PushBobbinMahjongList,
		WinInfos:        &winInfos,
		RoomId:          request.Uuid,
		RoomPlayerInfo:  []*pb.RoomPlayerInfo{},
	}

	// 对玩家进行结算 ************  start
	// 1.每个玩家都进行判断，将其输赢赋值给
	//common.LogDebug("待结算玩家数：", len(request.PlayerInfo))
	//common.LogDebug("输赢：", winInfos.AllAreaWin)
	// 庄家索引
	var bankerIndex int
	// 庄家总输赢
	var BankerWinBalance int64 = 0
	// 遍历处理玩家输赢
	for v, k := range request.PlayerInfo {
		//当用户不在房间时
		if k.Uuid == "" {
			continue
		}
		//跳过庄家
		if k.Uuid == request.BankerUuid {
			bankerIndex = v
			continue
		}

		if k.PlayerRoomState == pb.PlayerRoomState_PlayerRoomStatePlay {
			//结算玩家输赢
			WinMoney, MsgErr, tempWinBalance, betAmount := obj.getWinMoney(k, winInfos, Commission, bankerOdds)
			if MsgErr != nil {
				common.LogError("PushPushBobbinSettle Drive getWinMoney has err", MsgErr)
				return MsgErr
			}
			// 更新庄家输赢
			BankerWinBalance += tempWinBalance
			// 改变房间信息里玩家的金额(这里更新用户多扣的金币和输的金币）
			request.PlayerInfo[v].Balance += WinMoney + betAmount
			request.PlayerInfo[v].WinOrLose += WinMoney
			PushRoomState.RoomPlayerInfo = append(PushRoomState.RoomPlayerInfo, request.PlayerInfo[v])
			common.LogDebug("玩家信息：", k.Uuid, "下注区域：", k.PlayerBets, "牌型:", winInfos.AllAreaOdds, "输赢：", WinMoney, request.PlayerInfo[v].WinOrLose)
		}
	}
	// 2.庄家输赢
	BankerWinBalance = obj.Compensation(request, BankerWinBalance, bankerIndex)
	if request.BankerUuid != "systemBanker" {
		water := int64(0)
		// 计算庄家税收
		if BankerWinBalance > 0 {
			water = (BankerWinBalance / 100) * Commission
			BankerWinBalance -= water
		}
		// 更新庄家信息
		request.PlayerInfo[bankerIndex].Balance += BankerWinBalance
		request.PlayerInfo[bankerIndex].WinOrLose = BankerWinBalance
		request.PlayerInfo[bankerIndex].HundredWaterBill = common.AbsInt64(BankerWinBalance)
		request.PlayerInfo[bankerIndex].HundredCommission = water
		PushRoomState.RoomPlayerInfo = append(PushRoomState.RoomPlayerInfo, request.PlayerInfo[bankerIndex])
	}
	PushRoomState.BankerWinOrLose = BankerWinBalance

	common.RoomBroadcast(request, PushRoomState)
	//****************************  end

	// 更新血池
	var score int64
	for _, k := range request.PlayerInfo {
		if k.IsRobot || k.Uuid == "" {
			continue
		}

		score -= k.WinOrLose + k.HundredCommission
	}

	MsgErr := common.BloodIncrease(score, request.GetGameType(), request.GetGameScene())
	if MsgErr != nil {
		common.LogError("PushBobbin Settle Drive Increase Blood has err:", MsgErr)
		return MsgErr
	}

	// 修改玩家真实的Money
	for _, onePlayerInfo := range request.GetPlayerInfo() {
		if onePlayerInfo.HundredWaterBill == 0 {
			continue
		}
		if onePlayerInfo.Uuid != "" {
			// 后面协程操作，为避免错误在此处提取金额
			addMoney := onePlayerInfo.GetGetBonus() + onePlayerInfo.GetWinOrLose()
			//common.LogDebug("同步金币：", onePlayerInfo.Account, "奖金：", onePlayerInfo.GetGetBonus(), "纯输赢：", onePlayerInfo.GetWinOrLose())
			go obj.saveMoney(request, onePlayerInfo, addMoney, winInfos)
		}
	}

	//下个状态
	request.NextRoomState = pb.RoomState_RoomStateReady
	request.DoTime = nowTime + int64(settleTime)
	return nil
}

// 金币结算
func (obj *PushBobbinSettle) getWinMoney(player *pb.RoomPlayerInfo, winInfo pb.PushBobbinWinInfo, Commission int64,
	bankerOdds int64) (int64, *pb.ErrorMessage, int64, int64) {
	var WinBalance float64 = 0
	// 庄家输赢值
	var TempWinBalance int64 = 0
	// 个人流水值
	var WaterNum int64 = 0
	// 个人抽水值
	var CommissionNum int64 = 0
	// 多扣金币数
	var betAmount int64 = 0
	// 遍历下注区域
	for key, bet := range player.PlayerBets {
		if bet == 0 {
			continue
		}
		//盈利值
		var winBalance int64 = 0
		betAmount += bet * (MaxOddsNumber - 1)
		//赔率
		odds := winInfo.AllAreaOdds[key]
		if winInfo.AllAreaWin[key] == 1 {
			// 赢
			winBalance = bet * odds
			tempWinBalance := winBalance + bet
			// 累加庄家输赢
			TempWinBalance -= winBalance
			// 减去抽水
			var water = math.Ceil(float64(winBalance) / 100 * float64(Commission))
			waterInt := int64(water) % 10
			//抽水最低10，必须是10的倍数，个位向上取整
			if waterInt != 0 {
				water = water - float64(waterInt) + 10
			}
			WinBalance += float64(tempWinBalance) - water
			// 更新流水（减去扣税）
			WaterNum += int64(float64(winBalance) - water)
			// 更新抽水
			CommissionNum += int64(water)
		} else {
			// 按照庄家赔率算
			winBalance = bet * bankerOdds
			// 更新庄家的输赢(抽水在外面统一处理）
			TempWinBalance += winBalance
			//输了的时候要判断是不是翻倍下注，如果是的话要按照翻倍扣除下注金额
			if winBalance > bet {
				//下注的翻倍区域，特殊处理
				WinBalance -= float64(winBalance - bet)
			}
			// 更新流水
			WaterNum += winBalance
		}
	}
	// 更新玩家抽水信息
	player.HundredWaterBill = WaterNum
	player.HundredCommission = CommissionNum
	//结算金额向下取整
	return int64(WinBalance), nil, TempWinBalance, betAmount
}

// getPokerType 获得牌型和倍数
// 参数：牌组
// 返回值：牌组类型，牌组倍数，rpc错误
func getMahjongType(roomInfo *pb.RoomInfo, pokers []int64) (pb.PushBobbinMahjongType, int64, *pb.ErrorMessage) {
	mahjongType := GetMahjongType(pokers)
	typeNum := strconv.Itoa(int(mahjongType))

	configOddsName := "OddsDigit" + typeNum
	oddsStr := common.GetRoomConfig(roomInfo, configOddsName)
	oddsNum, err := strconv.Atoi(oddsStr)
	if err != nil {
		common.LogError("getMahjongType strconv.Atoi(oddsStr) has err", err, configOddsName)
		return pb.PushBobbinMahjongType_PushBobbinMahjongType_None, int64(1), common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return mahjongType, int64(oddsNum), nil
}

// 分配庄家输赢
func (obj *PushBobbinSettle) Compensation(roomInfo *pb.RoomInfo, bankerWinBalance int64, bankerIndex int) int64 {
	loseAmount := bankerWinBalance
	if bankerWinBalance < 0 && roomInfo.BankerUuid != "systemBanker" && bankerWinBalance > roomInfo.PlayerInfo[bankerIndex].Balance {
		common.LogError("推筒子庄家金币不足结算:", roomInfo.PlayerInfo[bankerIndex].Balance, bankerWinBalance)
		loseAmount = -roomInfo.PlayerInfo[bankerIndex].Balance
	}
	// 根据比值分钱
	for key, val := range roomInfo.PlayerInfo {
		if val.WinOrLose <= 0 || val.Uuid == roomInfo.BankerUuid {
			continue
		}
		ratioMark := float64(val.WinOrLose) / float64(bankerWinBalance)
		// 计算取整
		myAmount := math.Floor(ratioMark * float64(loseAmount))
		tempNum := roomInfo.PlayerInfo[key].WinOrLose - int64(myAmount)
		if tempNum > 1 {
			// 不够赔
			roomInfo.PlayerInfo[key].WinOrLose -= tempNum
			roomInfo.PlayerInfo[key].Balance -= tempNum
			common.LogError("出现不够赔的情况，用户：", roomInfo.PlayerInfo[key].Account, "少赔金额:", tempNum)
		}
	}
	return loseAmount
}

// 同牌型比较输赢a是庄家(返回true则a大 ，false则a小)
func (obj *PushBobbinSettle) getCompareType(a []int64, b []int64) bool {
	// 升序排序
	listNumA := QuickSort(a)
	listNumB := QuickSort(b)
	//  点数相同，且点数不为0时，比双方牌中大的那张牌，如果相同，则庄家胜；若双方都为0点，则庄家胜。
	// 比较牌数
	if listNumA[1] > listNumB[1] {
		return true
	} else if listNumA[1] == listNumB[1] {
		return true
	} else {
		return false
	}
}

// 推送金币变动
func (obj *PushBobbinSettle) saveMoney(roomInfo *pb.RoomInfo, onePlayer *pb.RoomPlayerInfo, addMoney int64, winInfos pb.PushBobbinWinInfo) {
	taskConfig := &pb.TaskConfig{}
	taskConfig.TaskType = pb.TaskType_Task_PlayGame
	taskConfig.GameType = roomInfo.GetGameType()
	taskConfig.GameScene = roomInfo.GetGameScene()
	taskConfig.TaskNum = 1
	cateGoryList := strings.Split(common.GetRoomConfig(roomInfo, "CateGory"), ",")
	for _, oneCateGoryStr := range cateGoryList {
		oneCateGoryInt, err := strconv.Atoi(oneCateGoryStr)
		if err != nil {
			common.LogError("PushBobbinSettle saveMoney oneCateGoryStr has err", oneCateGoryStr, err)
		}
		taskConfig.GameCateGoryType = append(taskConfig.GetGameCateGoryType(), pb.GameCateGoryType(oneCateGoryInt))
	}
	afterBalance, msgErr := common.ChangePlayerInfoAfterGameEnd(onePlayer.GetUuid(), addMoney, taskConfig, pb.ResourceChangeReason_PushBobbinSettleGold)
	if msgErr != nil {
		return
	}
	//common.LogDebug(onePlayer.Account, onePlayer.Balance, afterBalance, addMoney)
	if afterBalance != onePlayer.Balance {
		common.LogError("PushBobbinSettle saveMoney has err: afterBalance != onePlayer.Balance", afterBalance, onePlayer.Balance, "输赢：", onePlayer.WinOrLose)
	}
	userBalanceChangePush := &pb.PushUserBalanceChange{}
	userBalanceChangePush.UserId = onePlayer.GetUuid()
	userBalanceChangePush.Balance = afterBalance
	common.RoomBroadcast(roomInfo, userBalanceChangePush)
}

//判断庄家是否是机器人
func (obj *PushBobbinSettle) isBankerIsRobot(playerList []*pb.RoomPlayerInfo, bankerId string) bool {
	if bankerId == "systemBanker" {
		return true
	}
	for _, k := range playerList {
		if k.Uuid == bankerId {
			if k.IsRobot {
				return true
			} else {
				return false
			}
		}
	}
	return true
}