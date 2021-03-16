package common

import (
	pb "gameServer-demo/src/grpc"
	"strconv"
	"strings"
	// "fmt"
)

// GetRoomConfig 获得房间的某项配置
func GetRoomConfig(roomInfo *pb.RoomInfo, name string) string {
	/*config := Configer.GetGameConfig(roomInfo.GetGameType(), roomInfo.GetGameScene(), name)
	if config == nil {
		return ""
	}
	return config.GetValue()*/
	for _, oneConfig := range roomInfo.GetConfig() {
		if oneConfig.GetName() == name {
			return oneConfig.GetValue()
		}
	}
	return ""
}

// UpdateRoomConfig 更新房间的某项配置
func UpdateRoomConfig(roomInfo *pb.RoomInfo, name string, value string) {
	for _, oneConfig := range roomInfo.GetConfig() {
		if oneConfig.GetName() == name {
			oneConfig.Value = value
			return
		}
	}
	newConfig := &pb.GameConfig{}
	newConfig.Name = name
	newConfig.Value = value
	roomInfo.Config = append(roomInfo.GetConfig(), newConfig)
}

// CustomRoomConfig 自定义房间配置，需检测合法之后，同步
func CustomRoomConfig(roomInfo *pb.RoomInfo, name string, value string) *pb.ErrorMessage {
	switch name {
	case "Ante":
		anteInt, err := strconv.Atoi(value)
		if err != nil {
			LogError("CustomRoomConfig Ante Atoi has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_AnteConfigError, "")
		}
		if anteInt < 1000 || anteInt > 1000000 {
			LogError("CustomRoomConfig Ante anteInt out range")
			return GetGrpcErrorMessage(pb.ErrorCode_AnteConfigError, "")
		}
		UpdateRoomConfig(roomInfo, name, value)
	case "EnterGoldBean":
		enterGoldBeanInt, err := strconv.Atoi(value)
		if err != nil {
			LogError("CustomRoomConfig EnterGoldBean Atoi has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_EnterConfigError, "")
		}
		if enterGoldBeanInt < 1000 || enterGoldBeanInt > 100000000000 {
			LogError("CustomRoomConfig EnterGoldBean enterGoldBeanInt out range")
			return GetGrpcErrorMessage(pb.ErrorCode_EnterConfigError, "")
		}
		UpdateRoomConfig(roomInfo, name, value)
	case "MaxPlayer":
		maxPlayerInt, err := strconv.Atoi(value)
		if err != nil {
			LogError("CustomRoomConfig MaxPlayer Atoi has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_MaxPlayerConfigError, "")

		}
		if maxPlayerInt < 2 || maxPlayerInt > 9 {
			LogError("CustomRoomConfig MaxPlayer maxPlayerInt out range")
			return GetGrpcErrorMessage(pb.ErrorCode_MaxPlayerConfigError, "")
		}
		UpdateRoomConfig(roomInfo, name, value)
	case "PlayerStartNum": // 有几个玩家就可以开始游戏了
		playerStartNumInt, err := strconv.Atoi(value)
		if err != nil {
			LogError("CustomRoomConfig PlayerStartNum Atoi has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		if playerStartNumInt < 2 || playerStartNumInt > 9 {
			LogError("CustomRoomConfig PlayerStartNum playerStartNumInt out range")
			return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		UpdateRoomConfig(roomInfo, name, value)
	case "RoomAllPlayNum": // 房间可玩的总局数
		gameName := roomInfo.GetGameType().String()
		roomPlayStr := Configer.GetGlobal(gameName + "RoomPlayNum").GetValue()
		masterPayStr := Configer.GetGlobal(gameName + "MasterPayNum").GetValue()
		aAPayStr := Configer.GetGlobal(gameName + "AAPayNum").GetValue()
		roomPlayArr := strings.Split(roomPlayStr, ",")
		masterPayArr := strings.Split(masterPayStr, ",")
		aAPayArr := strings.Split(aAPayStr, ",")
		if len(roomPlayArr) != len(masterPayArr) || len(masterPayArr) != len(aAPayArr) {
			LogError("CustomRoomConfig RoomAllPlayNum config err")
			return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		wantIndex := IndexOf(roomPlayArr, value)
		if wantIndex == -1 {
			LogError("CustomRoomConfig RoomAllPlayNum invalid RoomAllPlayNum")
			return GetGrpcErrorMessage(pb.ErrorCode_RoomAllPlayNumConfigError, "")
		}
		roomPlayInt, err := strconv.Atoi(roomPlayArr[wantIndex])
		if err != nil {
			LogError("CustomRoomConfig RoomAllPlayNum Atoi roomPlayArr has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		masterPayInt, err := strconv.Atoi(masterPayArr[wantIndex])
		if err != nil {
			LogError("CustomRoomConfig RoomAllPlayNum Atoi masterPayArr has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		aAPayInt, err := strconv.Atoi(aAPayArr[wantIndex])
		if err != nil {
			LogError("CustomRoomConfig RoomAllPlayNum Atoi aAPayArr has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		roomInfo.RoomAllPlayNum = int32(roomPlayInt)
		roomInfo.MasterPayNum = int64(masterPayInt)
		roomInfo.AaPayNum = int64(aAPayInt)
	case "PayType":
		payTypeInt, err := strconv.Atoi(value)
		if err != nil {
			LogError("CustomRoomConfig PayType Atoi has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_PayTypeConfigError, "")
		}
		payType := pb.PayType(payTypeInt)
		if payType == pb.PayType_PayType_None {
			LogError("CustomRoomConfig PayType payType has err")
			return GetGrpcErrorMessage(pb.ErrorCode_PayTypeConfigError, "")
		}
		roomInfo.PayType = payType
	}
	return nil
}
