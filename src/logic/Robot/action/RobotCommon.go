package action

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"strings"
)

func init() {
}

// 获取投注
func getBetMoney(roomInfo *pb.RoomInfo, index int) int64 {
	// 获取筹码值
	chipsStr := common.GetRoomConfig(roomInfo, "Chips")
	chipsArrStr := strings.Split(chipsStr, ",")
	// 将筹码转成数字
	chipsArr := make([]int64, 0)
	for _, k := range chipsArrStr {
		chipMoney, err := strconv.Atoi(k)
		if err != nil {
			common.LogError("getBetMoney Atoi ChipMoney has err:", err)
			return 0
		}
		chipsArr = append(chipsArr, int64(chipMoney))
	}
	// 返回筹码值
	return chipsArr[index]
}

// 根据比例获取一个随机投注区域
func getArea(Weight []int32) int {
	areaIndex, err := common.GetRandomIndexByWeight(Weight)
	for err != nil {
		common.LogError("getArea has err", err)
		return 0
	}
	return areaIndex
}

// 检查是否庄家或者在庄家列表
func alreadyInRank(uuid string, roomInfo *pb.RoomInfo) bool {
	if uuid == roomInfo.BankerUuid {
		return true
	}
	for _, k := range roomInfo.Bankers {
		if k == uuid {
			return true
		}
	}
	return false
}
