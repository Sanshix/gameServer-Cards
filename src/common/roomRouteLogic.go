package common

import (
	"strconv"

	pb "gameServer-demo/src/grpc"
)

// 获取一条服务器驱动索引
// 参数：playerInfo 玩家信息，serverNum 服务器数量，generateNew 如果玩家不在房间内是否随即指定一条线路
// 返回：driverServerIndex 路线
func GetDriverServerIndex(playerInfo *pb.PlayerInfo, serverNum int, generateNew bool) string {
	var driverServerIndex string
	if playerInfo.GetRoomId() != "" {
		driverServerIndex = playerInfo.GetGameServerIndex()
	} else {
		if generateNew {
			driverServerIndex = strconv.Itoa(GetRandomNum(1, serverNum))
		}
	}
	return driverServerIndex
}

//玩家进入房间route的判断
//参数：playerInfo 玩家信息，serverNum 服务器数量，gameScene 游戏场次
//返回：driverServerIndex 路线，错误
func GameJoinRoomJudge(playerInfo *pb.PlayerInfo, serverNum int, joinRequest *pb.GameJoinRoomRequest, gameType pb.GameType) (string, *pb.ErrorMessage) {
	driverServerIndex := strconv.Itoa(GetRandomNum(1, serverNum))
	if playerInfo.GetRoomId() != "" {
		driverServerIndex = playerInfo.GetGameServerIndex()
	} else {
		// 不能只通过房间id加入
		if joinRequest.GetRoomCode() == "" && joinRequest.GetRoomUUID() != "" {
			LogError("roomRouteLogic GameJoinRoomJudge JoinRoom RoomCode is nil", gameType, joinRequest.GameScene)
			return "", GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		// 如果是通过房间码或者房间id加入，则这里不判断金额，在加入房间的逻辑里判断
		if joinRequest.GetRoomCode() == "" && joinRequest.GetRoomUUID() == "" {
			enterBalanceStr := Configer.GetGameConfig(gameType, joinRequest.GameScene, "EnterBalance")
			if enterBalanceStr == nil {
				LogError("roomRouteLogic GameJoinRoomJudge JoinRoom EnterBalance config == nil", gameType, joinRequest.GameScene)
				return "", GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
			}
			enterBalance, err := strconv.Atoi(enterBalanceStr.GetValue())
			if err != nil {
				LogError("roomRouteLogic GameJoinRoomJudge JoinRoom EnterBalance has err", err, gameType, joinRequest.GameScene, enterBalanceStr)
				return "", GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
			}
			if int64(enterBalance) > playerInfo.GetBalance() {
				return "", GetGrpcErrorMessage(pb.ErrorCode_LessThanEnterBalance, "")
			}
		}
	}
	return driverServerIndex, nil
}
