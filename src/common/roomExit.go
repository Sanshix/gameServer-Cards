package common

import (
	pb "gameServer-demo/src/grpc"
)

// GameDriverExitRoom 退出房间通用接口
// 参数：组件名，房间管理器，rpc附加消息
// 返回：rpc错误信息
func GameDriverExitRoom(rm *RoomManager, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage {
	uuid := extroInfo.GetUserId()
	if uuid == "" {
		LogError("GameDriverExitRoom uuid == nil")
		return GetGrpcErrorMessage(pb.ErrorCode_UserNotLogin, "")
	}

	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		return msgErr
	}

	msgErr = rm.ExitRoom(loadPlayerReply.GetPlayerInfo(), extroInfo)
	return msgErr
}

// GameExitSingleRoom 退出单人房间通用接口
// 参数：退出房间的请求
// 返回：退出房间的返回，rpc错误信息
func GameExitSingleRoom(exitRoomRequest *pb.GameExitRoomRequest, playerInfo *pb.PlayerInfo, extroInfo *pb.MessageExtroInfo, selfDataFunc func(playerInfo *pb.PlayerInfo) *pb.ErrorMessage) (*pb.GameExitRoomReply, *pb.ErrorMessage) {
	reply := &pb.GameExitRoomReply{}
	// 先判断玩家是否在其他游戏中
	if playerInfo.GetGameType() != pb.GameType_None && playerInfo.GetGameType() != exitRoomRequest.GetGameType() {
		LogError("GameExitSingleRoom PlayerInOtherGame", exitRoomRequest, playerInfo.GetGameType())
		return reply, GetGrpcErrorMessage(pb.ErrorCode_PlayerInOtherGame, "")
	}
	playerInfo.GameType = pb.GameType_None
	playerInfo.GameScene = 0
	playerInfo.RoomId = ""
	msgErr := selfDataFunc(playerInfo)
	if msgErr != nil {
		LogError("GameEixtSingleRoom selfDataFunc has err", msgErr)
		return reply, msgErr
	}
	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = false
	msgErr = Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
	if msgErr != nil {
		return nil, nil
	}
	return reply, nil
}
