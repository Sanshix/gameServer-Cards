package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"github.com/golang/protobuf/ptypes"
	"strconv"
	"time"
)

func init() {
	common.AllComponentMap["PushBobbinBet"] = &PushBobbinBet{}
}

// PushBobbinReady 推筒子游戏的下注组件，用于处理下注阶段的逻辑
type PushBobbinBet struct {
	base.Base
}

// LoadComponent 加载组件
func (obj *PushBobbinBet) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *PushBobbinBet) Start() {
	obj.Base.Start()
}

// 推筒子下注阶段的主驱动
func (obj *PushBobbinBet) Drive(request *pb.RoomInfo, _ *pb.MessageExtroInfo) (*pb.RoomInfo, *pb.ErrorMessage) {
	// 获取当前时间时间戳
	nowTime := time.Now().Unix()
	// 100ms 推送一次
	nowNanoTime := time.Now().UnixNano()
	if (nowNanoTime-request.LastPushBetTime)/1e6 > 100 && len(request.PushPushBobbinBetPushes) > 0 {
		realPushMsg := &pb.PushPushBobbinPlayerBets{
			RoomId:   request.Uuid,
			PushBets: request.PushPushBobbinBetPushes,
		}
		common.RoomBroadcast(request, realPushMsg)
		// 字段置零
		request.PushPushBobbinBetPushes = make([]*pb.PushPushBobbinPlayerBet, 0)
		request.LastPushBetTime = nowNanoTime
	}

	if request.NextRoomState != pb.RoomState_RoomStateBet {
		if nowTime < request.DoTime {
			request.MilliDoTime = nowNanoTime/1e6 + 100
			return request, nil
		}
		// 这个时候还有消息没推送就推送
		if len(request.RedBlackBetPushes) > 0 {
			realPushMsg := &pb.PushPushBobbinPlayerBets{
				PushBets: request.GetPushPushBobbinBetPushes(),
			}
			common.RoomBroadcast(request, realPushMsg)
			// 字段置零
			request.PushPushBobbinBetPushes = make([]*pb.PushPushBobbinPlayerBet, 0)
		}
		request.CurRoomState = pb.RoomState_RoomStateSettle
		request.NextRoomState = pb.RoomState_RoomStateSettle
		request.DoTime = nowTime
		return request, nil
	}

	// 获取下注阶段时长
	betTimeStr := common.GetRoomConfig(request, "BetTime")
	betTime, err := strconv.Atoi(betTimeStr)
	if err != nil {
		common.LogError("RoomReadyLogic GameReadyDiver betTimeStr has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	// 推送房间状态改变的信息
	pushRoomState := &pb.PushRoomStateChange{
		RoomId:            request.GetUuid(),
		BeforeState:       pb.RoomState_RoomStateDeal,
		AfterState:        pb.RoomState_RoomStateBet,
		AfterStateEndTime: nowTime + int64(betTime),
	}
	common.RoomBroadcast(request, pushRoomState)

	//下个状态
	request.NextRoomState = pb.RoomState_RoomStateSettle
	request.DoTime = nowTime + int64(betTime)
	request.MilliDoTime = nowNanoTime/1e6 + 100 //下注区别与其他 100ms驱动一次
	return request, nil
}

// 玩家下注(区域：012）
func (obj *PushBobbinBet) RequestPlayerBet(request *pb.Driver2GameLogicInfo, extroInfo *pb.MessageExtroInfo) (*pb.Driver2GameLogicInfo, *pb.ErrorMessage) {

	reply := &pb.Driver2GameLogicInfo{}
	//获取用户id
	uuid := extroInfo.GetUserId()
	if uuid == "" {
		common.LogError("PushBobbinBet RequestPlayerBet uuid == nil")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotInRoom, "")
	}

	//庄家不能下注
	if uuid == request.RoomInfo.BankerUuid {
		common.LogError("PushBobbinBet RequestPlayerBet BankerUuid can not bet")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_BankerCannotBet, "")
	}

	//必须是下注状态才能下注
	if request.RoomInfo.CurRoomState != pb.RoomState_RoomStateBet {
		common.LogError("PushBobbinBet RequestPlayerBet Room State not is Bet")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_NotInBetTime, "")
	}

	// 下注请求回复实体类
	realRequest := &pb.PushBobbinBetRequest{}
	realReply := &pb.PushBobbinBetReply{
		RoomId: request.RoomInfo.Uuid,
	}

	// 把解析的json字符串放入realRequest中
	err := ptypes.UnmarshalAny(request.GetMessage(), realRequest)
	if err != nil {
		common.LogError("PushBobbinBet RequestPlayerBet ptypes.UnmarshalAny has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	playerIndex := -1
	// 获取下注玩家索引
	for v, k := range request.RoomInfo.PlayerInfo {
		if k.GetUuid() == uuid {
			playerIndex = v
			break
		}
	}

	//如果这个时候玩家索引还是-1  ，说明玩家不再房间里面，这是错误的
	if playerIndex == -1 {
		common.LogError("PushBobbinBet RequestPlayerBet player not in room ,playerIndex = -1")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotInRoom, "")
	}

	// 获取翻倍最大倍数 -天尊
	maxOdds := common.GetRoomConfig(request.RoomInfo, "OddsDigit12")
	maxOddsInt, err := strconv.ParseInt(maxOdds, 10, 64)
	if err != nil {
		common.LogError("PushBobbinBet Drive Bull20Odds has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	betBalance := int64(0)
	// 计算本金
	tempCount := int64(0)
	// 获取总下注等额
	for _, bet := range request.RoomInfo.PlayerInfo[playerIndex].PlayerBets {
		if bet == 0 {
			continue
		}
		betBalance += bet * maxOddsInt
		tempCount += bet
	}

	// 获取当前下注金额
	addBetBalance := realRequest.BetBalance - tempCount
	// 机器人下注不用判断幂等
	if request.RoomInfo.PlayerInfo[playerIndex].IsRobot == true {
		addBetBalance = realRequest.BetBalance
	}

	if addBetBalance < 0 {
		common.LogError("RequestPlayerBet addBetBalance < 0")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_BetRequestInvalid, "")
	} else if addBetBalance == 0 {
		//封装并返回
		realReply.IsSuccess = false
		realReply.RoomId = request.RoomInfo.GetUuid()
		realReplyAny, err := ptypes.MarshalAny(realReply)
		if err != nil {
			common.LogError("RequestPlayerBet MarshalAny has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		reply.RoomInfo = request.RoomInfo
		reply.Message = realReplyAny
		return reply, nil
	}
	// 已扣金币
	tempBetGold := betBalance
	// 加上当前下注(下注数按照最大翻倍数算）
	betBalance += addBetBalance * maxOddsInt
	tempBalance := addBetBalance * maxOddsInt

	// 判断玩家身上的钱是否够这次下注的钱 注：这里只做判断，限红处才将钱减去
	if request.RoomInfo.PlayerInfo[playerIndex].Balance+tempBetGold < betBalance {
		common.LogError("推筒子玩家下注金额不足")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_PlayerBalanceNotEnough, "")
	}

	// 推筒子共用一组限红
	if tempBalance <= request.RoomInfo.MaxBetRatio[0] {
		// 1 减限红,同时加给总注,加玩家下注
		request.RoomInfo.MaxBetRatio[0] -= tempBalance
		// 将下注金额累加到玩家下注与总注
		request.RoomInfo.PlayerInfo[playerIndex].PlayerBets[realRequest.BetArea] += addBetBalance
		request.RoomInfo.AllBet[realRequest.BetArea] += addBetBalance

		//2 减去房间信息里面玩家新增下注的金额 -- 最后结算才将金额从玩家表扣除
		request.RoomInfo.PlayerInfo[playerIndex].Balance -= tempBalance
		request.RoomInfo.PlayerInfo[playerIndex].WinOrLose -= addBetBalance

		//3 将下注成功的玩家状态改变成游戏中
		request.RoomInfo.PlayerInfo[playerIndex].PlayerRoomState = pb.PlayerRoomState_PlayerRoomStatePlay

		realReply.IsSuccess = true
	} else {
		common.LogError("推筒子玩家下注超出限红", realRequest.BetArea)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_BetRatioNotEnough, "")
	}
	//封装并返回
	realReplyAny, err := ptypes.MarshalAny(realReply)
	if err != nil {
		common.LogError("PushBobbinBet RequestPlayerBet MarshalAny has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.RoomInfo = request.RoomInfo
	reply.Message = realReplyAny

	// 将下注-房间广播
	pushMsg := &pb.PushPushBobbinPlayerBet{
		AllBet:        request.RoomInfo.AllBet,
		Uuid:          uuid,
		MaxBetRatio:   request.RoomInfo.MaxBetRatio,
		RoomId:        request.RoomInfo.GetUuid(),
		PlayerBet:     request.RoomInfo.PlayerInfo[playerIndex].PlayerBets,
		BetArea:       realRequest.BetArea,
		CurrentBet:    addBetBalance,
		PlayerBalance: request.RoomInfo.PlayerInfo[playerIndex].Balance,
	}

	// 字段没有就创建
	if request.RoomInfo.PushPushBobbinBetPushes == nil {
		request.RoomInfo.PushPushBobbinBetPushes = make([]*pb.PushPushBobbinPlayerBet, 0)
	}
	request.RoomInfo.PushPushBobbinBetPushes = append(request.RoomInfo.PushPushBobbinBetPushes, pushMsg)
	//common.RoomBroadcast(request.RoomInfo, pushMsg)
	//---------------判断限红
	maxbetRatio := request.RoomInfo.MaxBetRatio[0]
	//common.LogDebug("限红：", maxbetRatio)
	if maxbetRatio < 1000 {
		common.LogDebug("筹码已经达到庄家限红，直接开牌结算")
		request.RoomInfo.DoTime = time.Now().Unix()
	}
	//----------------------
	return reply, nil
}

//玩家上庄
func (obj *PushBobbinBet) RequestUpBanker(request *pb.Driver2GameLogicInfo, extroInfo *pb.MessageExtroInfo) (*pb.Driver2GameLogicInfo, *pb.ErrorMessage) {
	reply := &pb.Driver2GameLogicInfo{}
	//获取用户id
	uuid := extroInfo.GetUserId()
	if uuid == "" {
		common.LogError("PushBobbinBet RequestUpBanker uuid == nil")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotInRoom, "")
	}

	// 上庄请求回复实体类
	realRequest := &pb.PushBobbinUpBankerRequest{}
	realReply := &pb.PushBobbinUpBankerReply{}

	// 把解析的json字符串放入realRequest中
	err := ptypes.UnmarshalAny(request.GetMessage(), realRequest)
	if err != nil {
		common.LogError("PushBobbinBet RequestUpBanker ptypes.UnmarshalAny has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 获取上庄最小金额，上庄玩家列表最大长度
	userBankerMoneyStr := common.GetRoomConfig(request.RoomInfo, "UserBankerMoney")
	userBankerMoney, err := strconv.Atoi(userBankerMoneyStr)
	if err != nil {
		common.LogError("PushBobbinBet RequestUpBanker userBankerMoney has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	bankersLengthStr := common.GetRoomConfig(request.RoomInfo, "BankersLength")
	bankersLength, err := strconv.Atoi(bankersLengthStr)
	if err != nil {
		common.LogError("PushBobbinBet RequestUpBanker bankersLength has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	playerIndex := -1
	// 获取玩家在房间中的索引
	for v, k := range request.RoomInfo.PlayerInfo {
		if k.Uuid == uuid {
			playerIndex = v
			break
		}
	}

	// 索引还是-1 说明玩家不在房间
	if playerIndex == -1 {
		common.LogError("PushBobbinBet RequestUpBanker player not in room")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotInRoom, "")
	}

	// 判断上庄玩家是否已经在申请列表里
	for _, k := range request.RoomInfo.Bankers {
		if k == uuid {
			common.LogError("PushBobbinBet RequestUpBanker player already in Bankers,uuid = ", uuid)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_PlayerAlreadyInBanker, "")
		}
	}

	// 判断上庄玩家是否是庄家
	if uuid == request.RoomInfo.BankerUuid {
		common.LogError("PushBobbinBet RequestUpBanker uuid is Banker,cant up Bankers")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_PlayerAlreadyInBanker, "")
	}

	// 判断上庄玩家金额够否
	if request.RoomInfo.PlayerInfo[playerIndex].Balance < int64(userBankerMoney) {
		common.LogError("PushBobbinBet RequestUpBanker player Balance is not enough,uuid = ", uuid, " balance = ", request.RoomInfo.PlayerInfo[playerIndex].Balance)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_PlayerBalanceNotEnough, "")
	}

	// 判断上庄玩家列表是否有空位
	if len(request.RoomInfo.Bankers) >= bankersLength {
		common.LogError("PushBobbinBet RequestUpBanker bankers length >= ", bankersLength)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_BankersIsFull, "")
	}

	// 将用户加入到申请庄家列表，将玩家状态改变成游戏中
	request.RoomInfo.Bankers = append(request.RoomInfo.Bankers, uuid)
	request.RoomInfo.PlayerInfo[playerIndex].PlayerRoomState = pb.PlayerRoomState_PlayerRoomStatePlay
	realReply.IsSuccess = true
	common.LogDebug("庄家列表新增：", request.RoomInfo.Bankers)
	//封装并返回
	realReplyAny, errs := ptypes.MarshalAny(realReply)
	if errs != nil {
		common.LogError("PushBobbinBet RequestPlayerBet MarshalAny has err", errs)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.RoomInfo = request.RoomInfo
	reply.Message = realReplyAny

	// 广播现在庄家申请队列
	pushMsg := &pb.PushPushBobbinChangeBankers{
		RoomId:  request.RoomInfo.GetUuid(),
		Bankers: request.RoomInfo.Bankers,
	}
	common.RoomBroadcast(request.RoomInfo, pushMsg)

	return reply, nil
}

//玩家下庄
func (obj *PushBobbinBet) RequestDownBanker(request *pb.Driver2GameLogicInfo, extroInfo *pb.MessageExtroInfo) (*pb.Driver2GameLogicInfo, *pb.ErrorMessage) {
	reply := &pb.Driver2GameLogicInfo{}
	//获取用户id
	uuid := extroInfo.GetUserId()
	if uuid == "" {
		common.LogError("PushBobbinBet RequestDownBanker uuid == nil")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotInRoom, "")
	}

	// 下庄请求回复实体类
	realRequest := &pb.PushBobbinDownBankerRequest{}
	realReply := &pb.PushBobbinDownBankerReply{}

	// 把解析的json字符串放入realRequest中
	err := ptypes.UnmarshalAny(request.GetMessage(), realRequest)
	if err != nil {
		common.LogError("PushBobbinBet RequestDownBanker ptypes.UnmarshalAny has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 1.先查看用户是不是 在 庄家申请列表
	var BankersIndex = -1
	for v, k := range request.RoomInfo.Bankers {
		if k == uuid {
			BankersIndex = v
		}
	}
	// 在就将他删除+广播
	if BankersIndex != -1 {
		request.RoomInfo.Bankers = append(request.RoomInfo.Bankers[:BankersIndex], request.RoomInfo.Bankers[BankersIndex+1:]...)
		realReply.IsSuccess = true
		tempInfo := common.GetRoomPlayerInfo(request.RoomInfo, uuid)
		if tempInfo != nil {
			tempInfo.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateFree
		}
		pushMsg := &pb.PushPushBobbinChangeBankers{
			RoomId:  request.RoomInfo.GetUuid(),
			Bankers: request.RoomInfo.Bankers,
		}
		common.RoomBroadcast(request.RoomInfo, pushMsg)
	}

	// 2.如果玩家是庄家,设置庄家申请了下庄,在每次游戏free阶段,将庄家改变
	if uuid == request.RoomInfo.BankerUuid {
		request.RoomInfo.DownBankerQuest = true
		realReply.IsSuccess = true
	}
	//封装并返回
	realReplyAny, err := ptypes.MarshalAny(realReply)
	if err != nil {
		common.LogError("PushBobbinBet RequestDownBanker MarshalAny has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.RoomInfo = request.RoomInfo
	reply.Message = realReplyAny

	return reply, nil
}

//获取房间输赢记录
func (obj *PushBobbinBet) RequestRoomWinLogs(request *pb.Driver2GameLogicInfo, extroInfo *pb.MessageExtroInfo) (*pb.Driver2GameLogicInfo, *pb.ErrorMessage) {
	reply := &pb.Driver2GameLogicInfo{}
	//获取用户id
	uuid := extroInfo.GetUserId()
	if uuid == "" {
		common.LogError("PushBobbinBet RequestRoomWinLogs uuid == nil")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotInRoom, "")
	}

	// 回复实体类
	realRequest := &pb.PushBobbinGetRoomWinLogRequest{}
	realReply := &pb.PushBobbinGetRoomWinLogReply{}

	// 把解析的json字符串放入realRequest中
	err := ptypes.UnmarshalAny(request.GetMessage(), realRequest)
	if err != nil {
		common.LogError("PushBobbinBet RequestRoomWinLogs ptypes.UnmarshalAny has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	winRecord := &pb.PushBobbinWinRecord{
		PushBobbinWinInfos: request.RoomInfo.PushBobbinWinInfos,
		RoomId:             request.RoomInfo.Uuid,
	}
	realReply.WinRecord = winRecord
	realReply.IsSuccess = true
	//封装并返回
	realReplyAny, err := ptypes.MarshalAny(realReply)
	if err != nil {
		common.LogError("PushBobbinBet RequestRoomWinLogs MarshalAny has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.RoomInfo = request.RoomInfo
	reply.Message = realReplyAny

	return reply, nil
}

//获取房间玩家列表
func (obj *PushBobbinBet) RequestGetRoomPlayerList(request *pb.Driver2GameLogicInfo, extroInfo *pb.MessageExtroInfo) (*pb.Driver2GameLogicInfo, *pb.ErrorMessage) {
	reply := &pb.Driver2GameLogicInfo{}
	//获取用户id
	uuid := extroInfo.GetUserId()
	if uuid == "" {
		common.LogError("PushBobbinBet RequestGetRoomPlayerList uuid == nil")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotInRoom, "")
	}

	// 回复实体类
	realRequest := &pb.PushBobbinGetRoomPlayersRequest{}
	realReply := &pb.PushBobbinGetRoomPlayersReply{}

	// 把解析的json字符串放入realRequest中
	err := ptypes.UnmarshalAny(request.GetMessage(), realRequest)
	if err != nil {
		common.LogError("PushBobbinBet RequestGetRoomPlayerList ptypes.UnmarshalAny has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	realReply.PlayerInfo = request.RoomInfo.PlayerInfo
	realReply.IsSuccess = true
	//封装并返回
	realReplyAny, err := ptypes.MarshalAny(realReply)
	if err != nil {
		common.LogError("PushBobbinBet RequestGetRoomPlayerList MarshalAny has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.RoomInfo = request.RoomInfo
	reply.Message = realReplyAny

	return reply, nil
}
