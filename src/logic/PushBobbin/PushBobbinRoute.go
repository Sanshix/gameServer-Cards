package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"strconv"
)

func init() {
	common.AllComponentMap["PushBobbinRoute"] = &PushBobbinRoute{}
}

// PushBobbinRoute 推筒子游戏的功能中转组件，其他服务通过这个组件中转推筒子协议到具体逻辑组件中
type PushBobbinRoute struct {
	base.Base
}

// LoadComponent 加载组件
func (obj *PushBobbinRoute) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *PushBobbinRoute) Start() {
	obj.Base.Start()

	// 需要的配置进行模版初始化
	initGlobleConfigNameArr := []string{
		"PushBobbinServerNum",
	}
	for _, oneGlobleConfigName := range initGlobleConfigNameArr {
		oneConfig, ok := common.GlobleConfigTemp[oneGlobleConfigName]
		if !ok {
			panic("PushBobbinRoute initGlobleConfigNameArr has err")
		}
		common.Configer.SetGlobal(oneConfig, false)
	}
}

// Do 中转协议的具体逻辑(玩家的请求操作通过do路由到driver）
func (obj *PushBobbinRoute) Do(request *pb.PushBobbinDoRequest, extroInfo *pb.MessageExtroInfo) (proto.Message, *pb.ErrorMessage) {
	doType := request.GetDoType()
	uuid := extroInfo.GetUserId()
	if uuid == "" {
		common.LogError("PushBobbinRoute Do uuid == nil")
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotLogin, "")
	}
	// 选择一个driver线路
	originMessage := request.GetDoMessageContent()
	serverNumConfig := common.Configer.GetGlobal("PushBobbinServerNum")
	serverNumStr := serverNumConfig.GetValue()
	serverNum, err := strconv.Atoi(serverNumStr)
	if err != nil {
		common.LogError("PushBobbinRoute Do Atoi(serverNumStr) has err", serverNumStr)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	// 获取玩家信息
	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		return nil, msgErr
	}
	playerInfo := loadPlayerReply.GetPlayerInfo()

	var requestMessage proto.Message
	var replyMessage proto.Message
	//玩家线路
	var driverServerIndex string
	//组件名称
	var componentName string
	//方法名
	var methodName string
	//判断请求操作是否存在PushBobbinDoType中
	switch doType {
	//进入房间--与其他逻辑不同
	case pb.PushBobbinDoType_PushBobbin_JoinRoom:
		requestMessage = &pb.GameJoinRoomRequest{}
		replyMessage = &pb.GameJoinRoomReply{}

		err := ptypes.UnmarshalAny(originMessage, requestMessage)
		if err != nil {
			common.LogError("PushBobbinRoute Do joinRoom ptypes.UnmarshalAny has err", doType, uuid, err)
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		driverServerIndex, msgErr = common.GameJoinRoomJudge(playerInfo, serverNum, requestMessage.(*pb.GameJoinRoomRequest), pb.GameType_PushBobbin)
		if msgErr != nil {
			return nil, msgErr
		}

		methodName = "RequestJoinRoom"
	//退出房间
	case pb.PushBobbinDoType_PushBobbin_ExitRoom:
		requestMessage = &pb.GameExitRoomRequest{}
		replyMessage = &pb.GameExitRoomReply{}
		driverServerIndex = common.GetDriverServerIndex(playerInfo, serverNum, true)
		methodName = "RequestExitRoom"
	//玩家下注
	case pb.PushBobbinDoType_PushBobbin_PlayerBet:
		requestMessage = &pb.PushBobbinBetRequest{}
		replyMessage = &pb.PushBobbinBetReply{}
		driverServerIndex = common.GetDriverServerIndex(playerInfo, serverNum, true)
		methodName = "RequestPlayerBet"
	//玩家上庄
	case pb.PushBobbinDoType_PushBobbin_UpBanker:
		requestMessage = &pb.PushBobbinUpBankerRequest{}
		replyMessage = &pb.PushBobbinUpBankerReply{}
		driverServerIndex = common.GetDriverServerIndex(playerInfo, serverNum, true)
		methodName = "RequestUpBanker"
	//玩家下庄
	case pb.PushBobbinDoType_PushBobbin_DownBanker:
		requestMessage = &pb.PushBobbinDownBankerRequest{}
		replyMessage = &pb.PushBobbinDownBankerReply{}
		driverServerIndex = common.GetDriverServerIndex(playerInfo, serverNum, true)
		methodName = "RequestDownBanker"
	//获取输赢历史记录
	case pb.PushBobbinDoType_PushBobbin_GetRoomWinLog:
		requestMessage = &pb.PushBobbinGetRoomWinLogRequest{}
		replyMessage = &pb.PushBobbinGetRoomWinLogReply{}
		driverServerIndex = common.GetDriverServerIndex(playerInfo, serverNum, true)
		methodName = "RequestRoomWinLogs"
	//获取最新玩家列表
	case pb.PushBobbinDoType_PushBobbin_GetRoomPlayerList:
		requestMessage = &pb.PushBobbinGetRoomPlayersRequest{}
		replyMessage = &pb.PushBobbinGetRoomPlayersReply{}
		driverServerIndex = common.GetDriverServerIndex(playerInfo, serverNum, true)
		methodName = "RequestGetRoomPlayerList"

	default:
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	err = ptypes.UnmarshalAny(originMessage, requestMessage)
	if err != nil {
		common.LogError("PushBobbinRoute Do request ptypes.UnmarshalAny has err", doType, uuid, err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	componentName = "PushBobbinDriver" + driverServerIndex
	common.LogDebug("接受请求：", methodName)
	// common.Router.Call 都是路由到 driver,  driver 再分发给各个组件
	msgErr = common.Router.Call(componentName, methodName, requestMessage, replyMessage, extroInfo)
	if msgErr != nil {
		common.LogError(msgErr)
		return nil, msgErr
	}
	common.LogDebug("返回信息：", replyMessage)
	return replyMessage, nil
}

// SyncRoomPlayerInfo 同步房间玩家信息
func (obj *PushBobbinRoute) SyncRoomPlayerInfo(request *pb.SyncRoomPlayerInfo, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	reply := &pb.EmptyMessage{}
	driverServerIndex := request.GetServerIndex()
	methodName := "SyncRoomPlayerInfo"
	componentName := "PushBobbinDriver" + driverServerIndex
	msgErr := common.Router.Call(componentName, methodName, request, reply, extroInfo)
	if msgErr != nil {
		return nil, msgErr
	}
	return reply, nil
}

// 刪除房間
func (obj *PushBobbinRoute) DelRoom(request *pb.DelRoomByGameTypeRequest, extraInfo *pb.MessageExtroInfo) (*pb.DelRoomByGameTypeReply, *pb.ErrorMessage) {
	reply := &pb.DelRoomByGameTypeReply{}
	methodName := "DelRoom"
	componentName := "PushBobbinDriver" + request.GetServerIndex()
	msgErr := common.Router.Call(componentName, methodName, request, reply, extraInfo)
	if msgErr != nil {
		return reply, msgErr
	}
	return reply, nil
}
