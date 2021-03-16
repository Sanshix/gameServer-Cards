package common

import (
	"bytes"
	pb "gameServer-demo/src/grpc"
	"strconv"

	"github.com/golang/protobuf/proto"
)

func getRoomCodeInfo(roomCode string) (*pb.RoomCodeInfo, *pb.ErrorMessage) {
	extroInfo := &pb.MessageExtroInfo{}
	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = RedisRoomCodeInfoTable
	redisRequest.Key = roomCode
	redisReply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "HGetByte", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		LogError("GetRoomCodeInfo HGetByte has err", msgErr)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_InValidRoomCode, "")
	}
	if redisReply.GetValueByte() == nil {
		return nil, nil
	}

	reply := &pb.RoomCodeInfo{}
	err := proto.Unmarshal(redisReply.GetValueByte(), reply)
	if err != nil {
		LogError("GetRoomCodeInfo Unmarshal has err", err)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// GetRoomCodeInfo 通过房间码获得房间码信息
func GetRoomCodeInfo(roomCode string) (*pb.RoomCodeInfo, *pb.ErrorMessage) {
	extraInfo := &pb.MessageExtroInfo{}
	roomCodeInfoMutex, err := Locker.MessageLock(MessageLockRoomCodeInfo, extraInfo, "RoomCodeInfo")
	if err != nil {
		LogError("GetRoomCodeInfo MessageLock has err", err)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockRoomCodeInfo, extraInfo, "RoomCodeInfo", roomCodeInfoMutex)
	return getRoomCodeInfo(roomCode)
}

// CreateRoomCodeInfo 创建房间码信息并返回房间码
func CreateRoomCodeInfo(roomCardInfo *pb.RoomCodeInfo) (string, *pb.ErrorMessage) {
	extraInfo := &pb.MessageExtroInfo{}
	roomCodeInfoMutex, err := Locker.MessageLock(MessageLockRoomCodeInfo, extraInfo, "RoomCodeInfo")
	if err != nil {
		LogError("CreateRoomCodeInfo MessageLock has err", err)
		return "", GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockRoomCodeInfo, extraInfo, "RoomCodeInfo", roomCodeInfoMutex)

	roomCode := ""
	// 尝试10次
	for index := 0; index < 10; index++ {
		var buffer bytes.Buffer
		roomCodeTemp := ""
		for index := 0; index < 6; index++ {
			randomNum := GetRandomNum(0, 9)
			buffer.WriteString(strconv.Itoa(randomNum))
		}
		roomCodeTemp = buffer.String()
		roomCodeInfoTemp, msgErr := getRoomCodeInfo(roomCodeTemp)
		if msgErr != nil {
			continue
		}
		if roomCodeInfoTemp == nil {
			roomCode = roomCodeTemp
			break
		}
	}
	if roomCode == "" {
		LogError("CreateRoomCodeInfo roomCode create fail")
		return "", GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	roomCardInfo.RoomCode = roomCode
	roomCardInfoByte, err := proto.Marshal(roomCardInfo)
	if err != nil {
		LogError("CreateRoomCodeInfo Marshal has err", err)
		return "", GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	extroInfo := &pb.MessageExtroInfo{}
	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = RedisRoomCodeInfoTable
	redisRequest.Key = roomCode
	redisRequest.ValueByte = roomCardInfoByte
	redisReply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "HSetByte", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		LogError("GetRoomCodeInfo HSetByte has err", msgErr)
		return "", GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return roomCode, nil
}

// DeleteRoomCodeInfo 删除房间码
func DeleteRoomCodeInfo(roomCode string) *pb.ErrorMessage {
	extraInfo := &pb.MessageExtroInfo{}
	roomCodeInfoMutex, err := Locker.MessageLock(MessageLockRoomCodeInfo, extraInfo, "RoomCodeInfo")
	if err != nil {
		LogError("DeleteRoomCodeInfo MessageLock has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(MessageLockRoomCodeInfo, extraInfo, "RoomCodeInfo", roomCodeInfoMutex)

	roomDelRequest := &pb.RedisMessage{}
	roomDelRequest.Table = RedisRoomCodeInfoTable
	roomDelRequest.ValueStringArr = []string{roomCode}
	roomDelReply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "HDel", roomDelRequest, roomDelReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		return msgErr
	}
	return nil
}
