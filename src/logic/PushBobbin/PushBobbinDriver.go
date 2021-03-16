package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
)

func init() {
	common.AllComponentMap["PushBobbinDriver"] = &PushBobbinDriver{}
}

// PushBobbinDriver 推筒子游戏的房间管理组件，负责处理玩家请求操作
type PushBobbinDriver struct {
	base.Base
	rm *common.RoomManager
}

var maxRoomNumCfgName = "PushBobbinMaxRoomNumOneServer"

// LoadComponent 加载组件
func (obj *PushBobbinDriver) LoadComponent(config *common.OneComponentConfig, componentName string) {
	// 这是一个多线路组件
	obj.Base.LoadComponent(config, componentName)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *PushBobbinDriver) Start() {
	obj.Base.Start()

	initGlobleConfigNameArr := []string{
		maxRoomNumCfgName,
	}
	err := common.InitGlobleConfigTemp(initGlobleConfigNameArr)
	if err != nil {
		panic(err)
	}
	//将模板参数保存到内存中
	common.InitGameConfigTemp(common.PushBobbinGameConfigTemp, pb.GameType_PushBobbin)

	obj.rm = new(common.RoomManager)
	tableName := common.GetRoomRedisName(pb.GameType_PushBobbin, common.ServerIndex)
	obj.rm.InitRoomManager(pb.GameType_PushBobbin, tableName, obj.DriveRoom)
	//从redis里恢复房间数据（会导致配置不更新, 要刷新房间需要清空redis）
	err = obj.rm.ReLoadRooms()
	if err != nil {
		panic(err)
	}
	obj.rm.ReStartRooms()
	common.LogDebug("❤推筒子服务初始化❤")
}

// DriveRoom 这个方法提供房间驱动的具体逻辑
//(room公共驱动调用到这里，通过这个函数根据房间状态调用相应的房间逻辑)
func (obj *PushBobbinDriver) DriveRoom(roomInfo *pb.RoomInfo) *pb.RoomInfo {
	extraInfo := &pb.MessageExtroInfo{}
	afterRoomInfo := &pb.RoomInfo{}

	if roomInfo.CurRoomState == pb.RoomState_RoomStateBankChange {
		roomInfo.CurRoomState = pb.RoomState_RoomStateLocation
		roomInfo.NextRoomState = pb.RoomState_RoomStateLocation
	}
	// 根据状态获取需要调用的组件名
	componentName := (*obj.Base.Config)[pb.RoomState_name[int32(roomInfo.CurRoomState)]]
	if componentName == "" {
		common.LogError("PushBobbin DriveRoom get componentName has empty")
		return nil
	}
	//common.LogDebug("房间切换状态：", componentName, roomInfo.DoTime)
	msgErr := common.Router.Call(componentName, "Drive", roomInfo, afterRoomInfo, extraInfo)
	if msgErr != nil {
		common.LogError("PushBobbin DriveRoom call ", componentName, " Drive has err", msgErr)
		return afterRoomInfo
	}
	return afterRoomInfo
}

// RequestJoinRoom 加入房间逻辑
func (obj *PushBobbinDriver) RequestJoinRoom(request *pb.GameJoinRoomRequest, extroInfo *pb.MessageExtroInfo) (*pb.GameJoinRoomReply, *pb.ErrorMessage) {
	reply := &pb.GameJoinRoomReply{}
	roomInfo, msgErr := common.GameDriverJoinRoom(request, maxRoomNumCfgName, obj.rm, extroInfo)
	if msgErr != nil {
		common.LogError("PushBobbinDriver gameDriverJoinRoom get RoomInfo has error:", msgErr)
		return nil, msgErr
	}
	// 赋值玩家下注区域
	for v, k := range roomInfo.PlayerInfo {
		if k.Uuid == extroInfo.UserId {
			roomInfo.PlayerInfo[v].PlayerBets = make([]int64, 3)
			break
		}
	}
	reply.RoomInfo = roomInfo
	return reply, msgErr
}

// RequestExitRoom 退出房间逻辑
func (obj *PushBobbinDriver) RequestExitRoom(request *pb.GameExitRoomRequest, extroInfo *pb.MessageExtroInfo) (*pb.GameExitRoomReply, *pb.ErrorMessage) {
	reply := &pb.GameExitRoomReply{}
	msgErr := common.GameDriverExitRoom(obj.rm, extroInfo)
	return reply, msgErr
}

// RequestPlayerBet 玩家下注逻辑
func (obj *PushBobbinDriver) RequestPlayerBet(request *pb.PushBobbinBetRequest, extroInfo *pb.MessageExtroInfo) (*pb.PushBobbinBetReply, *pb.ErrorMessage) {
	reply := &pb.PushBobbinBetReply{}
	msgErr := common.GameDriverDo("PushBobbinBet", "RequestPlayerBet", request, reply, obj.rm, extroInfo)
	if msgErr != nil {
		return reply, msgErr
	}
	return reply, nil
}

// RequestPlayerBet 玩家上庄逻辑
func (obj *PushBobbinDriver) RequestUpBanker(request *pb.PushBobbinUpBankerRequest, extroInfo *pb.MessageExtroInfo) (*pb.PushBobbinUpBankerReply, *pb.ErrorMessage) {
	reply := &pb.PushBobbinUpBankerReply{}
	msgErr := common.GameDriverDo("PushBobbinBet", "RequestUpBanker", request, reply, obj.rm, extroInfo)
	if msgErr != nil {
		return reply, msgErr
	}
	return reply, nil
}

// RequestPlayerBet 玩家下庄逻辑
func (obj *PushBobbinDriver) RequestDownBanker(request *pb.PushBobbinDownBankerRequest, extroInfo *pb.MessageExtroInfo) (*pb.PushBobbinDownBankerReply, *pb.ErrorMessage) {
	reply := &pb.PushBobbinDownBankerReply{}
	msgErr := common.GameDriverDo("PushBobbinBet", "RequestDownBanker", request, reply, obj.rm, extroInfo)
	if msgErr != nil {
		return reply, msgErr
	}
	return reply, nil
}

// RequestRoomWinInfos 获取房间历史记录
func (obj *PushBobbinDriver) RequestRoomWinLogs(request *pb.PushBobbinGetRoomWinLogRequest, extroInfo *pb.MessageExtroInfo) (*pb.PushBobbinGetRoomWinLogReply, *pb.ErrorMessage) {
	reply := &pb.PushBobbinGetRoomWinLogReply{}
	msgErr := common.GameDriverDo("PushBobbinBet", "RequestRoomWinLogs", request, reply, obj.rm, extroInfo)
	if msgErr != nil {
		return reply, msgErr
	}
	return reply, nil
}

// RequestGetRoomPlayerList 获取玩家列表
func (obj *PushBobbinDriver) RequestGetRoomPlayerList(request *pb.PushBobbinGetRoomPlayersRequest, extroInfo *pb.MessageExtroInfo) (*pb.PushBobbinGetRoomPlayersReply, *pb.ErrorMessage) {
	reply := &pb.PushBobbinGetRoomPlayersReply{}
	msgErr := common.GameDriverDo("PushBobbinBet", "RequestGetRoomPlayerList", request, reply, obj.rm, extroInfo)
	if msgErr != nil {
		return reply, msgErr
	}
	return reply, nil
}

// SyncRoomPlayerInfo 同步玩家房间玩家信息
func (obj *PushBobbinDriver) SyncRoomPlayerInfo(request *pb.SyncRoomPlayerInfo, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	reply := &pb.EmptyMessage{}
	msgErr := common.GameDriverSyncRoomPlayerInfo(request, obj.rm, extroInfo)
	return reply, msgErr
}

// DelRoom删除某房间
func (obj *PushBobbinDriver) DelRoom(request *pb.DelRoomByGameTypeRequest, extraInfo *pb.MessageExtroInfo) (*pb.DelRoomByGameTypeReply, *pb.ErrorMessage) {
	reply := &pb.DelRoomByGameTypeReply{}
	// 根据uuid删除房间
	err := obj.rm.DeleteRoom(request.GetRoomUUID(), true)
	if err != nil {
		common.LogError("driver delRoom has err:", request.GameType, err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}
