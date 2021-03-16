package common

import (
	pb "gameServer-demo/src/grpc"

	"github.com/golang/protobuf/proto"
)

// RoomBroadcast 房间广播
func RoomBroadcast(roomInfo *pb.RoomInfo, pushMessage proto.Message) {
	for _, oneInfo := range roomInfo.GetPlayerInfo() {
		if oneInfo.GetUuid() == "" {
			continue
		}
		Pusher.Push(pushMessage, oneInfo.GetUuid())
	}
}

// PlayerStateChangeBroadcast 玩家状态改变推送接口封装
func PlayerStateChangeBroadcast(roomInfo *pb.RoomInfo, uuid string, beforeState pb.PlayerRoomState, afterState pb.PlayerRoomState) {
	pushMessage := &pb.PushPlayerStateChange{}
	pushMessage.RoomId = roomInfo.GetUuid()
	pushMessage.UserId = uuid
	pushMessage.BeforeState = beforeState
	pushMessage.AfterState = afterState
	RoomBroadcast(roomInfo, pushMessage)
}

//向当前房间的所有人推送消息
//参数：向指定用户推送的消息，向其他用户推送的消息，指定用户，房间信息
//返回：err
//使用方法：1.当房间广播的内容不分用户时，传入uuid为""，房间所有玩家只传pushMsg1
//		   2.当房间广播为需要区分的内容时，指定uuid传pushMsg1，其余为pushMsg2
//		   例如：摸牌广播，摸牌用户知道自己手牌，其他用户知道别人摸牌，但不知道是什么牌，就用方法2
func PushRoom(pushMsg1 proto.Message, pushMsg2 proto.Message, uid string, roomInfo *pb.RoomInfo) *pb.ErrorMessage {
	//获取房间所有的用户
	RoomUserId := make([]string, 0)
	for _, k := range roomInfo.PlayerInfo {
		RoomUserId = append(RoomUserId, k.Uuid)
	}
	//当用户没在房间时，，中止且报错
	var Judge bool
	for _, k := range RoomUserId {
		if uid == k {
			Judge = true
		}
	}
	if uid != "" && !Judge {
		LogError("RoomBroadcast PushRoom Room without this uid")
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "Room without this uid")
	}
	if len(RoomUserId) == 0 {
		LogError("RoomBroadcast PushRoom No users in the room")
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "No users in the room")
	}
	//当用户uid为空时,向房间所有用户推送消息pushMsg1
	if uid == "" {
		for _, OneUuid := range RoomUserId {
			Pusher.Push(pushMsg1, OneUuid)
		}
		return nil
	}
	//当用户uid不空时,只给uid推送pushMsg1，其余用户推送pushMsg2
	if uid != "" {
		for _, OneUuid := range RoomUserId {
			if OneUuid == uid {
				Pusher.Push(pushMsg1, OneUuid)
			} else {
				Pusher.Push(pushMsg2, OneUuid)
			}
		}
		return nil
	}
	return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
}
