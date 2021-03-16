package common

import (
	pb "gameServer-demo/src/grpc"
	"strconv"
	"strings"
	"time"
)

// 房间相关的公用方法

// 判断玩家在房间的索引 不在返回-1
func GetPlayerIndex(uuid string, roomInfo *pb.RoomInfo) (r int) {
	r = -1
	for k, v := range roomInfo.PlayerInfo {
		if v.GetUuid() == uuid {
			r = k
			break
		}
	}
	return r
}

// 玩家是否在庄家队列
// 参数：玩家Uuid,房间roomInfo
// 返回：是否在bool
func PlayerIsInBankers(uuid string, roomInfo *pb.RoomInfo) bool {
	for _, k := range roomInfo.Bankers {
		if k == uuid {
			return true
		}
	}
	return false
}

// 判断玩家钱够不够与玩家是否在房间里面
// 参数：玩家Uuid,最小金额minBalance,房间roomInfo
// 返回：是否在bool
func PlayerMoneyEnoughOrInRoom(uuid string, minBalance int64, roomInfo *pb.RoomInfo) bool {
	for _, k := range roomInfo.PlayerInfo {
		if k.Uuid == uuid && k.WaitKick == pb.RoomSeatsChangeReason_RoomSeatsChangeReason_None {
			if k.Balance >= minBalance {
				return true
			}
		}
	}
	return false
}

// GetFirstJoinRoomPlayer 获得第一个在时间上加入房间的人的uuid
func GetFirstJoinRoomPlayer(roomInfo *pb.RoomInfo) string {
	player := ""
	joinRoomTime := int64(0)
	for _, k := range roomInfo.GetPlayerInfo() {
		if k.GetUuid() == "" {
			continue
		}
		if joinRoomTime == 0 || k.GetJoinRoomTime() < joinRoomTime {
			player = k.GetUuid()
			joinRoomTime = k.GetJoinRoomTime()
			continue
		}
	}
	return player
}

// 下座位  抢座模式用
// 参数：房间roomInfo,玩家索引playerIndex,桌子isTable
func DownSeat(roomInfo *pb.RoomInfo, playerIndex int) bool {
	var newIndex int

	if playerIndex >= 8 {
		return false
	}

	for i := 8; i < len(roomInfo.PlayerInfo); i++ {
		if roomInfo.PlayerInfo[i].Uuid == "" {
			newIndex = i
			break
		}
	}

	// 有空位 就互换 + 将保险箱金额回收
	if newIndex != 0 {
		roomInfo.PlayerInfo[newIndex], roomInfo.PlayerInfo[playerIndex] = roomInfo.PlayerInfo[playerIndex], roomInfo.PlayerInfo[newIndex]
		roomInfo.PlayerInfo[playerIndex].Balance += roomInfo.PlayerInfo[playerIndex].SafeMoney
		roomInfo.PlayerInfo[playerIndex].SafeMoney = 0
		// 没空位,就追加 - 同时置空座位 + 将保险箱金额回收
	} else {
		roomInfo.PlayerInfo = append(roomInfo.PlayerInfo, roomInfo.PlayerInfo[playerIndex])
		roomInfo.PlayerInfo[newIndex] = &pb.RoomPlayerInfo{}
		roomInfo.PlayerInfo[len(roomInfo.PlayerInfo)-1].Balance += roomInfo.PlayerInfo[len(roomInfo.PlayerInfo)-1].SafeMoney
		roomInfo.PlayerInfo[len(roomInfo.PlayerInfo)-1].SafeMoney = 0
	}
	return true
}

// 检测座位上的人数 打璇0-7
func CheckTableNum(roomInfo *pb.RoomInfo, tableNum int) int {
	var reply int
	for _, v := range roomInfo.PlayerInfo[:tableNum] {
		if v.Uuid != "" {
			reply++
		}
	}
	return reply
}

// GetRoomRedisName 根据游戏类型和server index获取redis表名
func GetRoomRedisName(gameType pb.GameType, serverIndex string) string {
	return RedisRoomsTable + gameType.String() + ":" + serverIndex
}

// CheckRoomExsit 判断房间是否真的不存在，在redis中
func CheckRoomExsit(gameType pb.GameType, serverIndex string, roomUUID string) (bool, *pb.ErrorMessage) {
	roomRedisName := GetRoomRedisName(gameType, serverIndex)
	roomInfoRequest := &pb.RedisMessage{}
	roomInfoRequest.Table = roomRedisName
	roomInfoRequest.Key = roomUUID
	roomInfoReply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "HGetByte", roomInfoRequest, roomInfoReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("CheckRoomExsit HGetByte has err", roomRedisName, roomUUID, msgErr)
		return false, msgErr
	}
	if roomInfoReply.GetValueByte() == nil {
		return false, nil
	}
	return true, nil
}

// 判断房间是否是百人场
func JudgeRoomIsHundred(roomInfo *pb.RoomInfo) bool {

	CateGoryStr := GetRoomConfig(roomInfo, "CateGory")
	typesStr := strings.Split(CateGoryStr, ",")
	for _, k := range typesStr {
		if k == "3" {
			return true
		}
	}
	return false
}

// GetRoomPlayerInfo 获得房间的某个玩家的信息
func GetRoomPlayerInfo(roomInfo *pb.RoomInfo, uuid string) *pb.RoomPlayerInfo {
	for _, oneInfo := range roomInfo.GetPlayerInfo() {
		if oneInfo.GetUuid() == uuid {
			return oneInfo
		}
	}
	return nil
}

// CheckPushDoTimeInReady 检测房间PushDoTimeInReady的推送信息
func CheckPushDoTimeInReady(roomInfo *pb.RoomInfo, isNewRound bool) {
	if roomInfo.GetCurRoomState() != pb.RoomState_RoomStateReady {
		return
	}
	playerStartNumStr := GetRoomConfig(roomInfo, "PlayerStartNum")
	playerStartNum, err := strconv.Atoi(playerStartNumStr)
	if err == nil {
		validPlayerNum := 0
		for _, onePlayer := range roomInfo.GetPlayerInfo() {
			if onePlayer.GetUuid() != "" {
				validPlayerNum++
			}
		}
		readyTimeStr := GetRoomConfig(roomInfo, "ReadyTime")
		readyTimeNum, err := strconv.Atoi(readyTimeStr)
		if err != nil {
			readyTimeNum = 5
		}
		doTime := roomInfo.GetDoTime()
		if validPlayerNum > 1 {
			if validPlayerNum <= playerStartNum {
				doTime = int64(time.Now().Unix() + int64(readyTimeNum))
			} else {
				doTime = roomInfo.GetDoTime()
			}
			if isNewRound == true {
				doTime = int64(time.Now().Unix() + int64(readyTimeNum))
			}
			pushDoTimeInReady := &pb.PushDoTimeInReady{
				RoomId: roomInfo.Uuid,
				DoTime: doTime,
			}
			RoomBroadcast(roomInfo, pushDoTimeInReady)
		}
	}
}
