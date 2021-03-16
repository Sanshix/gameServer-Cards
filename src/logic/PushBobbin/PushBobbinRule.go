package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"sort"
)

func init() {
	common.AllComponentMap["PushBobbinRule"] = &PushBobbinRule{}
}

// PushBobbinRule 推筒子游戏的算法组件，负责中转牛牛游戏的牌型计算等
type PushBobbinRule struct {
	base.Base
}

// LoadComponent 加载组件
func (obj *PushBobbinRule) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *PushBobbinRule) Start() {
	obj.Base.Start()
}

// GetmahjongType 获得牌组的类型
func GetMahjongType(cardHeap []int64) pb.PushBobbinMahjongType {
	// 默认0
	pushBobbinMahjongType := pb.PushBobbinMahjongType_PushBobbinMahjongType_None
	// 对手牌排序
	cardHeap = QuickSort(cardHeap)
	// 天尊（一对白板）
	if cardHeap[0] == 0 && cardHeap[1] == 0 {
		pushBobbinMahjongType = pb.PushBobbinMahjongType(12)
		return pushBobbinMahjongType
	}
	// 天杠（一个2一个8）
	if (cardHeap[0] == 2 && cardHeap[1] == 8) || (cardHeap[0] == 8 && cardHeap[1] == 2) {
		pushBobbinMahjongType = pb.PushBobbinMahjongType(11)
		return pushBobbinMahjongType
	}
	// 豹子（对子）
	if cardHeap[0] == cardHeap[1] {
		pushBobbinMahjongType = pb.PushBobbinMahjongType(10)
		return pushBobbinMahjongType
	}
	// 普通点数
	singular := (cardHeap[0] + cardHeap[1]) % 10
	pushBobbinMahjongType = pb.PushBobbinMahjongType(singular)
	return pushBobbinMahjongType
}

//选择排序/* 从小到大 */
func QuickSort(slice []int64) []int64 {

	for i := 0; i < len(slice)-1; i++ {

		for k := i + 1; k < len(slice); k++ {

			if slice[i] > slice[k] {

				slice[i], slice[k] = slice[k], slice[i]

			}
		}
	}
	return slice
}

// 获取指定的推筒子牌型（牌组权重范围内随机取）
func PushBobbinGetAssignMahjong(roomInfo *pb.RoomInfo, bunko int, weight int,
	cardHeap []int64) ([]int64, int, []int64) {
	cardType := 0
	min := 0
	switch bunko {
	case 1:
		// 庄家赢牌
		weight = 10
		min = 6
	case 2:
		// lose 获取输的牌
		weight -= 1
	case 3:
		// 闲家拿大牌
		min = weight + 1
		weight = 10
	case 4:
		// 随机牌，但是限制特殊牌型,和庄家一样的牌型区间
		if weight == 10 {
			min = 7
		}
	}
	// 获得牌型后拿指定牌型的牌
	var replyCard []int64
	replyCardHeap := cardHeap
	for i := 0; i < len(cardHeap); i++ {
		tempCards := append([]int64{}, cardHeap[i])
		// 获取新的临时牌堆
		tempHeap := append([]int64{}, cardHeap[:i]...)
		tempHeap = append(tempHeap, cardHeap[i+1:]...)
		tempState := 1
		for x := 0; x < len(tempHeap); x++ {
			tempReplyCard := append(tempCards, tempHeap[x])
			mahjongType, _, msgErr := getMahjongType(roomInfo, tempReplyCard)
			if msgErr != nil {
				common.LogError("GetAssignCards has err", msgErr)
				continue
			}
			// 牌型匹配
			if int(mahjongType) >= min && int(mahjongType) <= weight {
				replyCard = append([]int64{}, cardHeap[i], tempHeap[x])
				// 获取新的牌堆
				replyCardHeap = append(tempHeap[:x], tempHeap[x+1:]...)
				cardType = int(mahjongType)
				tempState = 2
				break
			}
		}
		if tempState == 2 {
			break
		}
	}
	//匹配不到相应牌型时随机
	if replyCard == nil {
		replyCard = append(replyCard, cardHeap[0], cardHeap[1])
		replyCardHeap = replyCardHeap[2:]
		common.LogDebug("无法匹配相应牌型：", min, weight)
	}

	return replyCard, cardType, replyCardHeap
}

// 返回下注数最多的两个区域（用于控制的情况下避免每次全赢或者全输）
func OrderBets(list []*pb.RoomPlayerInfo, bankerIsRobot bool) (int, int) {
	// 同化翻倍下注数
	tempBets := make(map[int]int64, 4)
	// 是否有机器人下注
	robotMark := false
	noRobotMark := false
	for _, v := range list {
		if bankerIsRobot == true && v.IsRobot == true {
			continue
		}
		for k, vv := range v.PlayerBets {
			if vv == 0 {
				continue
			}
			if v.IsRobot == true {
				robotMark = true
			} else {
				noRobotMark = true
			}

			tempBets[k] += vv
		}
	}
	// 过滤玩家对玩家/ 机器人对机器人
	if len(tempBets) == 0 || (robotMark == false && bankerIsRobot == false) || (noRobotMark == false && bankerIsRobot == true) {
		// 没人下注 都随机
		return 8, 8
	}
	// 转成slice
	type kv struct {
		Key   int
		Value int64
	}
	var ss []kv
	for k, v := range tempBets {
		ss = append(ss, kv{k + 1, v})
	}
	// 降序排序
	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})
	if len(tempBets) >= 2 {
		return ss[0].Key, ss[1].Key
	} else {
		// 随机处理
		temp := common.GetRandomNum(1, 4)
		return ss[0].Key, temp
	}
}

// 获取推筒子牌堆 （1~9筒和白板）
func getPushBobbinMahjongHeap() []int64 {
	var pokers []int64
	for x := int64(0); x <= 9; x++ {
		pokers = append(pokers, x, x, x, x)
	}
	return pokers
}
