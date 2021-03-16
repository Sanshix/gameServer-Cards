package common

import (
	pb "gameServer-demo/src/grpc"
	"strconv"
)

// GameDriverJoinRoom 加入房间通用接口
// 参数：组件名，想要加入的场次，服务器所能容纳的最大房间数，房间管理器，rpc附加消息
// 返回：房间信息，rpc错误信息
func GameDriverJoinRoom(joinRoomRequest *pb.GameJoinRoomRequest, maxRoomConfigName string, rm *RoomManager, extroInfo *pb.MessageExtroInfo) (*pb.RoomInfo, *pb.ErrorMessage) {
	return GameDriverJoinRoomWithCreateRoomFunc(joinRoomRequest, maxRoomConfigName, rm, extroInfo, nil)
}

// GameDriverJoinRoomWithCreateRoomFunc 加入房间通用接口
// 参数：组件名，想要加入的场次，服务器所能容纳的最大房间数，房间管理器，rpc附加消息
// 返回：房间信息，rpc错误信息
func GameDriverJoinRoomWithCreateRoomFunc(
	joinRoomRequest *pb.GameJoinRoomRequest,
	maxRoomConfigName string,
	rm *RoomManager,
	extroInfo *pb.MessageExtroInfo,
	createRoomFunc func(room *pb.RoomInfo) *pb.ErrorMessage) (*pb.RoomInfo, *pb.ErrorMessage) {
	uid := extroInfo.GetUserId()
	if uid == "" {
		LogError("CommonJoinRoom uuid == nil")
		return nil, GetGrpcErrorMessage(pb.ErrorCode_UserNotLogin, "")
	}

	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		return nil, msgErr
	}
	// 通过邀请码加入
	if joinRoomRequest.GetRoomCode() != "" && joinRoomRequest.GetRoomUUID() == "" {
		roomCodeInfo, msgErr := GetRoomCodeInfo(joinRoomRequest.GetRoomCode())
		if msgErr != nil {
			return nil, msgErr
		}
		joinRoomRequest.RoomUUID = roomCodeInfo.GetRoomUUID()
	}

	// 先尝试加入房间
	roomInfo, msgErr := rm.JoinRoom(loadPlayerReply.GetPlayerInfo(), joinRoomRequest, extroInfo)
	if msgErr != nil && msgErr.GetCode() != pb.ErrorCode_NotJoinRoom {
		return nil, msgErr
	}
	if roomInfo != nil {
		return roomInfo, nil
	}
	// 如果是通过房间id加入的，没有能加的房间内就返回错误
	if joinRoomRequest.GetRoomUUID() != "" {
		LogError("CommonJoinRoom RoomNotExist", joinRoomRequest.GetRoomUUID())
		return nil, GetGrpcErrorMessage(pb.ErrorCode_RoomNotExist, "")
	}
	// 如果是通过房间id加入的，没有能加的房间内就返回错误
	playInfo := loadPlayerReply.GetPlayerInfo()
	if playInfo.GetRoomId() != "" {
		realRoomExsit, msgErr := CheckRoomExsit(playInfo.GetGameType(), playInfo.GetGameServerIndex(), playInfo.GetRoomId())
		if msgErr != nil {
			LogError("CommonJoinRoom CheckRoomExsit hse err", msgErr)
			return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		if realRoomExsit == true {
			LogError("CommonJoinRoom user in other game", playInfo.GetUuid())
			return nil, GetGrpcErrorMessage(pb.ErrorCode_PlayerInOtherGame, "")
		}
		LogError("CommonJoinRoom RoomNotExist2", joinRoomRequest.GetRoomUUID())
		loadPlayerReply.GetPlayerInfo().GameType = pb.GameType_None
		loadPlayerReply.GetPlayerInfo().GameScene = 0
		loadPlayerReply.GetPlayerInfo().GameServerIndex = ""
		loadPlayerReply.GetPlayerInfo().RoomId = ""
		savePlayerRequest := &pb.SavePlayerRequest{}
		savePlayerRequest.PlayerInfo = loadPlayerReply.GetPlayerInfo()
		savePlayerRequest.ForceSave = false
		msgErr = Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
		if msgErr != nil {
			return nil, msgErr
		}

		return nil, GetGrpcErrorMessage(pb.ErrorCode_RoomNotExist, "")
	}

	// 没有可加入的就先创建房间
	maxRoomNumConfig := Configer.GetGlobal(maxRoomConfigName)
	maxRoomNumStr := maxRoomNumConfig.GetValue()
	maxRoomNum, err := strconv.Atoi(maxRoomNumStr)
	if err != nil {
		LogError("CommonJoinRoom Atoi(serverNumStr) has err", maxRoomNumStr)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	msgErr = rm.CreateRoomWithFunc(maxRoomNum, joinRoomRequest, createRoomFunc)
	if msgErr != nil {
		LogError("CommonJoinRoom CreateRoom has err", msgErr)
		return nil, msgErr
	}
	// 再尝试加入
	roomInfo, msgErr = rm.JoinRoom(loadPlayerReply.GetPlayerInfo(), joinRoomRequest, extroInfo)
	if msgErr != nil {
		return nil, msgErr
	}
	if roomInfo != nil {
		return roomInfo, nil
	}

	return nil, GetGrpcErrorMessage(pb.ErrorCode_NotJoinRoom, "")
}
