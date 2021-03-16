package common

import (
	"errors"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	uuid "github.com/satori/go.uuid"
)

// RoomDoFunc 房间操作函数
// 参数：房间信息
// 返回值：修改后的房间信息，处理后的消息体，rpc错误
type RoomDoFunc func(roomInfo *pb.RoomInfo) (*pb.RoomInfo, proto.Message, *pb.ErrorMessage)

// RoomDriveFunc 房间驱动函数
// 参数：房间信息
// 返回值：修改后的房间信息
type RoomDriveFunc func(roomInfo *pb.RoomInfo) *pb.RoomInfo

// Room 房间的结构体，包括房间信息，房间锁等
type Room struct {
	roomInfo          *pb.RoomInfo
	rlock             sync.Mutex
	roomTimer         *time.Timer
	roomTimerStopChan chan bool
}

// RoomManager 房间的管理器，包括房间管理的一些基本操作，这个结构体应该仅被构建在游戏Driver服务上
type RoomManager struct {
	roomMap    map[string]*Room
	rmLock     sync.Mutex
	gameType   pb.GameType
	RedisTable string
	DriveFunc  func(roomInfo *pb.RoomInfo) *pb.RoomInfo
}

// InitRoomManager 初始化房间管理器
func (rm *RoomManager) InitRoomManager(gameType pb.GameType, redisTable string, driveFunc RoomDriveFunc) {
	rm.rmLock.Lock()
	defer rm.rmLock.Unlock()

	rm.roomMap = make(map[string]*Room)
	rm.gameType = gameType
	rm.DriveFunc = driveFunc
	rm.RedisTable = redisTable
	StartTimer(time.Minute*1, false, func() bool {
		rm.ClearRoomLoop()
		return true
	})
}

// ClearRoomLoop 清除房间的循环
func (rm *RoomManager) ClearRoomLoop() {
	rm.rmLock.Lock()
	defer rm.rmLock.Unlock()

	for roomID, room := range rm.roomMap {
		if room.IsDead() == true {

			// 抢座模式 返回玩家本金
			if CheckModeOpen(pb.GameMode_GameMode_Grab) {
				nowTime := time.Now().Unix()
				// 获取惩罚额度
				penaltyQuotaStr := GetRoomConfig(room.GetRoomInfo(), "PenaltyQuota")
				penaltyQuota, err := strconv.Atoi(penaltyQuotaStr)
				if err != nil {
					LogError("Room ClearRoomLoop died penaltyQuotaStr has err", err)
				}

				// 获取惩罚比例
				penaltyRatioStr := GetRoomConfig(room.GetRoomInfo(), "PenaltyRatio")
				penaltyRatio, err := strconv.Atoi(penaltyRatioStr)
				if err != nil {
					LogError("Room ClearRoomLoop died penaltyRatioStr has err", err)
					return
				}
				// 获取惩罚时间
				penaltyTimeStr := GetRoomConfig(room.GetRoomInfo(), "PenaltyTime")
				penaltyTime, err := strconv.Atoi(penaltyTimeStr)
				if err != nil {
					LogError("Room ClearRoomLoop died penaltyTimeStr has err", err)
					return
				}

				// 统计受惩罚的人 - 与输家总金额
				var allPunish, allLose int64
				for k, v := range room.GetRoomInfo().PlayerInfo {
					if v.Uuid != "" && v.AllSafeMoney > 0 {
						// 上回合的游戏时间也计入惩罚时间中
						if k < 8 && v.DownSeatRequest {
							v.PunishTime += nowTime - room.GetRoomInfo().RoundStartTime
						} else if k >= 8 && v.AllSafeMoney > 0 {
							v.PunishTime += nowTime - room.GetRoomInfo().RoundStartTime
						}

						// 惩罚时间 > 300s 且是赢家者
						// 将钱从safeMoney 扣 -- 不能大于safeMoney
						if v.PunishTime > int64(penaltyTime) && v.AllWinOrLose > int64(penaltyQuota) {
							onePunish := v.AllWinOrLose * int64(penaltyRatio) / 100
							if onePunish >= v.SafeMoney {
								v.SafeMoney -= onePunish
								allPunish += onePunish
							} else {
								allPunish += v.SafeMoney
								v.SafeMoney = 0
							}
						}
						// 输者计数
						if v.AllWinOrLose < 0 {
							allLose += AbsInt64(v.WinOrLose)
						}
					}
				}
				// 统计受惩罚的人 - 逃跑者
				for _, v := range room.GetRoomInfo().QPlayer {
					v.PunishTime += nowTime - room.GetRoomInfo().RoundStartTime // 上回合的游戏时间也计入惩罚时间中
					if v.PunishTime > int64(penaltyTime) && v.AllWinOrLose > int64(penaltyQuota) {
						onePunish := v.AllWinOrLose * int64(penaltyRatio) / 100
						if onePunish >= v.SafeMoney {
							v.SafeMoney -= onePunish
							allPunish += onePunish
						} else {
							allPunish += v.SafeMoney
							v.SafeMoney = 0
						}
					}
				}

				// 结算 在房间者
				for _, v := range room.GetRoomInfo().PlayerInfo {
					if v.Uuid != "" && v.AllSafeMoney > 0 {
						// 输赢 >= 0 正常结算
						if v.AllWinOrLose >= 0 {
							go ChangeOtherBalance(v.Uuid, v.SafeMoney, true, true, pb.ResourceChangeReason_DaXuanBackMoney)
						} else {
							v.SafeMoney += allPunish * AbsInt64(v.AllWinOrLose) / allLose
							go ChangeOtherBalance(v.Uuid, v.SafeMoney, true, true, pb.ResourceChangeReason_DaXuanBackMoney)
						}
					}
				}
				// 结算 逃跑者
				for _, v := range room.GetRoomInfo().QPlayer {
					// 输赢 >= 0 正常结算
					if v.AllWinOrLose >= 0 {
						go ChangeOtherBalance(v.PlayerUuid, v.SafeMoney, true, true, pb.ResourceChangeReason_DaXuanBackMoney)
					} else {
						v.SafeMoney += allPunish * AbsInt64(v.AllWinOrLose) / allLose
						go ChangeOtherBalance(v.PlayerUuid, v.SafeMoney, true, true, pb.ResourceChangeReason_DaXuanBackMoney)
					}
				}
				// 将所有玩家踢出
				for _, onePlayer := range room.GetRoomInfo().PlayerInfo {
					onePlayer.WaitKick = pb.RoomSeatsChangeReason_RoomSeatsChangeReason_RoomDisSolve
					onePlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateNone
				}
				room.Kick(true)
			}
			err := rm.DeleteRoom(roomID, false)
			if err != nil {
				continue
			}
		}
	}
	return
}

// DeleteRoom 删除房间
func (rm *RoomManager) DeleteRoom(roomID string, withLock bool) error {
	if withLock == true {
		rm.rmLock.Lock()
		defer rm.rmLock.Unlock()
	}
	if _, ok := rm.roomMap[roomID]; !ok {
		return errors.New("RoomManager DeleteRoom Redis rm.roomMap[roomID] not ok")
	}

	msgErr := rm.roomMap[roomID].Delete()
	if msgErr != nil {
		LogError("RoomManager DeleteRoom Delete has err", msgErr)
		return errors.New("RoomManager DeleteRoom Delete has err")
	}
	roomDelRequest := &pb.RedisMessage{}
	roomDelRequest.Table = rm.RedisTable
	roomDelRequest.ValueStringArr = []string{roomID}
	roomDelReply := &pb.RedisMessage{}
	msgErr = Router.Call("Redis", "HDel", roomDelRequest, roomDelReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("RoomManager DeleteRoom Redis HDel has err", msgErr)
		return errors.New("RoomManager DeleteRoom Redis HDel has err")
	}
	delete(rm.roomMap, roomID)

	return nil
}

// GetRoom 获得房间
func (rm *RoomManager) GetRoom(roomID string) *Room {
	rm.rmLock.Lock()
	defer rm.rmLock.Unlock()
	if _, ok := rm.roomMap[roomID]; ok {
		if rm.roomMap[roomID].roomInfo.GetDead() == true {
			return nil
		}
		return rm.roomMap[roomID]
	}
	return nil
}

//SyncRoomPlayerInfo 同步房间玩家信息
func (rm *RoomManager) SyncRoomPlayerInfo(request *pb.SyncRoomPlayerInfo) *pb.ErrorMessage {
	rm.rmLock.Lock()
	defer rm.rmLock.Unlock()
	LogDebug("RoomManager SyncRoomPlayerInfo in", request)
	if _, ok := rm.roomMap[request.GetRoomUUID()]; !ok {
		// 房间不存在返回修改成攻
		//return GetGrpcErrorMessage(pb.ErrorCode_RoomNotExist, "")
		return nil
	}
	return rm.roomMap[request.GetRoomUUID()].SyncRoomPlayerInfo(request)
}

// PayByOtherCallBack 代付回掉
func (rm *RoomManager) PayByOtherCallBack(request *pb.RoomPayByOtherInfo) *pb.ErrorMessage {
	rm.rmLock.Lock()
	defer rm.rmLock.Unlock()
	if _, ok := rm.roomMap[request.GetRoomUUID()]; !ok {
		// 房间不存在返回修改成攻
		//return GetGrpcErrorMessage(pb.ErrorCode_RoomNotExist, "")
		LogError("RoomManager PayByOtherCallBack room not exist", request)
		return nil
	}
	return rm.roomMap[request.GetRoomUUID()].PayByOtherCallBack(request)
}

// CreateRoom 创建房间
// 外部传入服务器所能容纳的最大房间数量,游戏场次，和房间对应的redis表名称,以及驱动函数
func (rm *RoomManager) CreateRoom(roomMaxInServer int, joinRoomRequest *pb.GameJoinRoomRequest) *pb.ErrorMessage {
	return rm.CreateRoomWithFunc(roomMaxInServer, joinRoomRequest, nil)
}

// CreateRoomWithFunc 创建房间
// 外部传入服务器所能容纳的最大房间数量,游戏场次，和房间对应的redis表名称,以及驱动函数
func (rm *RoomManager) CreateRoomWithFunc(roomMaxInServer int, joinRoomRequest *pb.GameJoinRoomRequest, createRoomFunc func(room *pb.RoomInfo) *pb.ErrorMessage) *pb.ErrorMessage {
	rm.rmLock.Lock()
	defer rm.rmLock.Unlock()

	if len(rm.roomMap) >= roomMaxInServer {
		LogError("RoomManager CreateRoomWithFunc too many rooms in server")
		return GetGrpcErrorMessage(pb.ErrorCode_TooManyRoomInServer, "")
	}
	gameKeyMap := Configer.GetGameConfigByGameTypeAndScene(rm.gameType, joinRoomRequest.GetGameScene())
	if gameKeyMap == nil {
		LogError("RoomManager GetGameConfigByGameTypeAndScene gameKeyMap == nil")
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	roomInfo := &pb.RoomInfo{}
	roomInfo.GhMahjongInRoom = &pb.GangHuaMahjongInRoom{}
	roomUUID := uuid.NewV4()
	roomUUIDStr := roomUUID.String()
	roomInfo.Uuid = roomUUIDStr
	roomInfo.CurRoomState = pb.RoomState_RoomStateReady
	roomInfo.NextRoomState = pb.RoomState_RoomStateReady
	roomInfo.DoTime = time.Now().Unix() + 5
	roomInfo.RedisTable = rm.RedisTable
	for _, oneConfig := range gameKeyMap.Map {
		roomInfo.Config = append(roomInfo.Config, oneConfig)
	}
	roomInfo.GameType = rm.gameType
	roomInfo.GameScene = joinRoomRequest.GetGameScene()
	roomInfo.CreateTime = time.Now().Unix()
	roomInfo.Dead = false
	roomInfo.GameServerIndex = ServerIndex
	roomInfo.IdelCount = 0
	if CheckModeOpen(pb.GameMode_GameMode_Grab) {
		roomInfo.RoomType = pb.RoomType_RoomType_SystemRoom
		roomInfo.DaXuanIsDiJiuWang = joinRoomRequest.IsDiJiuWang
	} else {
		roomInfo.RoomType = pb.RoomType_RoomType_Normal
	}

	roomInfo.RoomPlayNum = 1

	if nil != createRoomFunc {
		err := createRoomFunc(roomInfo)
		if err != nil {
			return err
		}
	}

	room := &Room{
		roomInfo: roomInfo,
	}
	rm.roomMap[roomUUIDStr] = room

	room.Drive(rm.DriveFunc)

	// 跑马灯房间创建推送
	go func() {
		if SelectComponentExist("HorseRaceLamp") {
			horseRaceLampWinerRequest := &pb.HorseRaceLampCreateRoomRequest{}
			horseRaceLampWinerRequest.Uuid = roomInfo.Uuid
			horseRaceLampReply := &pb.HorseRaceLampCreateRoomReply{}
			err := Router.Call("HorseRaceLamp", "PushHorseRaceLampCreateRoom", horseRaceLampWinerRequest, horseRaceLampReply, &pb.MessageExtroInfo{})
			if err != nil {
				LogError("Player PushHorseRaceLampCreateRoom call  has err", err)
			}
		}
	}()

	return nil
}

// ReLoadRooms 恢复房间
// 将所有本服务器需要管理的房间信息加载到内存中
func (rm *RoomManager) ReLoadRooms() error {
	rm.rmLock.Lock()
	defer rm.rmLock.Unlock()

	roomKeyRequest := &pb.RedisMessage{}
	roomKeyRequest.Table = rm.RedisTable
	roomKeyReply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "HKeys", roomKeyRequest, roomKeyReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		return errors.New("RoomManager ReLoadRooms Redis HKeys has err")
	}

	for _, oneRoomKey := range roomKeyReply.GetValueStringArr() {
		roomInfoRequest := &pb.RedisMessage{}
		roomInfoRequest.Table = rm.RedisTable
		roomInfoRequest.Key = oneRoomKey
		roomInfoReply := &pb.RedisMessage{}
		msgErr := Router.Call("Redis", "HGetByte", roomInfoRequest, roomInfoReply, &pb.MessageExtroInfo{})
		if msgErr != nil {
			return errors.New("RoomManager ReLoadRooms Redis HGetByte has err:" + oneRoomKey)
		}

		roomInfo := &pb.RoomInfo{}
		err := proto.Unmarshal(roomInfoReply.GetValueByte(), roomInfo)
		if err != nil {
			LogError("RoomManager ReLoadRooms proto.Unmarshal has err:", oneRoomKey)
			return err
		}
		if rm.gameType != roomInfo.GetGameType() {
			LogError("RoomManager ReLoadRooms proto.Unmarshal has err:", oneRoomKey)
			err = errors.New("RoomManager gameType not match with room")
			return err
		}

		room := &Room{
			roomInfo: roomInfo,
		}
		rm.roomMap[roomInfo.GetUuid()] = room
	}

	return nil
}

// ReStartRooms 恢复房间驱动
// 参数为驱动函数，驱动函数由外部实现
func (rm *RoomManager) ReStartRooms() {
	rm.rmLock.Lock()
	defer rm.rmLock.Unlock()

	for _, room := range rm.roomMap {
		room.Drive(rm.DriveFunc)
	}
	return
}

// JoinRoom 加入房间
func (rm *RoomManager) JoinRoom(playerInfo *pb.PlayerInfo, joinRoomRequest *pb.GameJoinRoomRequest, extroInfo *pb.MessageExtroInfo) (*pb.RoomInfo, *pb.ErrorMessage) {
	rm.rmLock.Lock()
	defer rm.rmLock.Unlock()

	for _, room := range rm.roomMap {
		roomType := room.roomInfo.GetRoomType()
		// 普通匹配房间才判断scene
		if roomType == pb.RoomType_RoomType_Normal ||
			roomType == pb.RoomType_RoomType_None {
			if room.roomInfo.GetGameScene() != joinRoomRequest.GetGameScene() {
				continue
			}
		}
		// 抢座模式
		if CheckModeOpen(pb.GameMode_GameMode_Grab) {
			// 只进指定房间 - 或者 - 进入playerInfo.roomId 里面的房间
			if room.roomInfo.Uuid == joinRoomRequest.RoomUUID || room.roomInfo.Uuid == playerInfo.RoomId {
				roomInfo, msgErr := room.TryJoinRoom(playerInfo, joinRoomRequest, extroInfo)
				if msgErr != nil {
					return nil, msgErr
				}
				if roomInfo != nil {
					return roomInfo, nil
				}
			} else {
				continue
			}
		} else {
			roomInfo, msgErr := room.TryJoinRoom(playerInfo, joinRoomRequest, extroInfo)
			if msgErr != nil {
				return nil, msgErr
			}
			if roomInfo != nil {
				return roomInfo, nil
			}
		}

	}

	return nil, GetGrpcErrorMessage(pb.ErrorCode_NotJoinRoom, "")
}

// ExitRoom 退出房间
func (rm *RoomManager) ExitRoom(playerInfo *pb.PlayerInfo, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage {
	roomID := playerInfo.GetRoomId()
	room := rm.GetRoom(roomID)

	if room == nil {
		return GetGrpcErrorMessage(pb.ErrorCode_RoomNotExist, "")
	}

	return room.TryExitRoom(playerInfo, extroInfo)
}

// Do 操作房间
func (rm *RoomManager) Do(playerInfo *pb.PlayerInfo, doFunc RoomDoFunc) (proto.Message, *pb.ErrorMessage) {

	roomID := playerInfo.GetRoomId()
	room := rm.GetRoom(roomID)

	if room == nil {
		return nil, GetGrpcErrorMessage(pb.ErrorCode_RoomNotExist, "")
	}

	reply, msgErr := room.Do(doFunc)
	if msgErr == nil {
		//延时drive，保证do所产生的推送优先到达
		//StartTimer(500*time.Millisecond, false, func() bool {
		room.Drive(rm.DriveFunc)
		//	return false
		//})
	}

	return reply, msgErr
}

// SyncRoomPlayerInfo 同步房间内玩家信息
func (r *Room) SyncRoomPlayerInfo(request *pb.SyncRoomPlayerInfo) *pb.ErrorMessage {
	r.rlock.Lock()
	defer r.rlock.Unlock()

	roomInfo := r.roomInfo
	for _, onePlayer := range roomInfo.GetPlayerInfo() {
		if onePlayer.GetUuid() == request.GetPlayerUUID() {
			afterBalance := onePlayer.GetBalance() + request.GetChangeBalance()
			if afterBalance < 0 {
				return GetGrpcErrorMessage(pb.ErrorCode_BalanceNotEnough, "")
			}
			onePlayer.Balance = afterBalance
			r.Save(false)
			return nil
		}
	}
	LogError("Room SyncRoomPlayerInfo player not in room", request.GetPlayerUUID())
	return GetGrpcErrorMessage(pb.ErrorCode_NotInRoom, "")
}

// PayByOtherCallBack 代付回掉
func (r *Room) PayByOtherCallBack(request *pb.RoomPayByOtherInfo) *pb.ErrorMessage {
	r.rlock.Lock()
	defer r.rlock.Unlock()
	LogDebug("Room PayByOtherCallBack in", request)

	if r.roomInfo.GetPayStatus() == pb.PayStatus_PayStatus_Wait {
		r.roomInfo.PayStatus = request.GetPayStatus()
		LogDebug("PayByOtherCallBack ok", request)
		r.Save(false)
		return nil
	}
	LogError("Room PayByOtherCallBack PayStatus err", r.roomInfo.GetPayStatus())
	return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
}

// Delete 删除房间
func (r *Room) Delete() *pb.ErrorMessage {
	r.rlock.Lock()
	defer r.rlock.Unlock()

	StopTimer(r.roomTimer, r.roomTimerStopChan)
	roomCode := r.roomInfo.GetRoomCode()
	if roomCode != "" {
		return DeleteRoomCodeInfo(roomCode)
	}
	return nil
}

// Drive 驱动房间
// 参数为驱动函数，驱动函数由外部实现
// 可以任意时刻手动驱动，手动驱动会重制定时器时间
func (r *Room) Drive(driveFunc RoomDriveFunc) {
	r.rlock.Lock()
	defer r.rlock.Unlock()

	nowTime := time.Now().Unix()
	//nextDriveTime := r.roomInfo.GetDoTime()
	StopTimer(r.roomTimer, r.roomTimerStopChan)
	/*if nextDriveTime > nowTime {
		timerTime := nextDriveTime - nowTime
		r.roomTimer, r.roomTimerStopChan = StartTimer(time.Duration(timerTime)*time.Second, false, func() bool {
			r.Drive(driveFunc)
			return false
		})
		return
	}*/
	//beforeDriveTime := r.roomInfo.GetDoTime()
	beforeState := r.roomInfo.GetCurRoomState()
	beforeNextState := r.roomInfo.GetNextRoomState()
	beforeDriveTime := nowTime
	// 获取毫秒级前一个 MilliDoTime
	beforeMilliTime := r.roomInfo.GetMilliDoTime()
	afterRoomInfo := driveFunc(r.roomInfo)
	if afterRoomInfo != nil && afterRoomInfo.GetUuid() != "" {
		r.roomInfo = afterRoomInfo
	}
	// 检测要被踢出的玩家，并且踢出
	r.Kick(false)
	// 获取毫秒级后一个 MilliDoTime
	afterMilliTime := r.roomInfo.GetMilliDoTime()
	afterDriveTime := r.roomInfo.GetDoTime()
	afterState := r.roomInfo.GetCurRoomState()
	afterNextState := r.roomInfo.GetNextRoomState()
	timerTime := afterDriveTime - beforeDriveTime

	// 如果驱动函数没有更改下次驱动时间的话，就自己延时5秒，避免死循环
	if timerTime <= 0 && beforeState == afterState && beforeNextState == afterNextState {
		timerTime = 5
	}

	if beforeState != afterState {
		r.roomInfo.LastRoomState = beforeState
		r.roomInfo.LastChangeStateTime = nowTime
	}

	// 如果驱动函数没有更改下次驱动时间的话，就自己延时5秒，避免死循环
	/*if afterDriveTime <= beforeDriveTime && beforeState == afterState {
		afterDriveTime = afterDriveTime + 5
		r.roomInfo.DoTime = afterDriveTime
		timerTime = 5
	}*/
	// 毫秒级被设定就执行毫秒级的
	// 否则：秒级设定
	if afterMilliTime > beforeMilliTime {
		r.roomTimer, r.roomTimerStopChan = StartTimer(time.Duration(afterMilliTime-beforeMilliTime)*time.Millisecond, false, func() bool {
			r.Drive(driveFunc)
			return false
		})
	} else {
		r.roomTimer, r.roomTimerStopChan = StartTimer(time.Duration(timerTime)*time.Second, false, func() bool {
			r.Drive(driveFunc)
			return false
		})
	}

	r.Save(false)
	return
}

// Do 操作房间
// 参数为操作函数，操作函数由外部实现
// 可以任意时刻手动驱动，手动驱动会重制定时器时间
func (r *Room) Do(doFunc RoomDoFunc) (proto.Message, *pb.ErrorMessage) {
	r.rlock.Lock()
	defer r.rlock.Unlock()

	afterRoomInfo, reply, msgErr := doFunc(r.roomInfo)
	if msgErr == nil && afterRoomInfo != nil {
		r.roomInfo = afterRoomInfo
		r.Save(false)
	}
	return reply, msgErr
}

// Save 保存一次房间信息
func (r *Room) Save(withLock bool) {
	if withLock {
		r.rlock.Lock()
		defer r.rlock.Unlock()
	}

	roomRequest := &pb.RedisMessage{}
	roomRequest.Table = r.roomInfo.GetRedisTable()
	roomRequest.Key = r.roomInfo.GetUuid()
	byteValue, err := proto.Marshal(r.roomInfo)
	if err != nil {
		LogError("Room Save proto.Marshal has err", err)
		return
	}
	roomRequest.ValueByte = byteValue
	roomReply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "HSetByte", roomRequest, roomReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("Room Save Redis HSetByte has err", msgErr)
		return
	}
}

// IsDead 房间是否死亡
func (r *Room) IsDead() bool {
	r.rlock.Lock()
	defer r.rlock.Unlock()

	return r.roomInfo.GetDead()
}

// GetRoomConfig 获得房间的某项配置
func (r *Room) GetRoomConfig(name string) string {
	roomInfo := r.roomInfo
	return GetRoomConfig(roomInfo, name)
}

// GetRoomInfo 获得房间的某项配置
func (r *Room) GetRoomInfo() *pb.RoomInfo {
	roomInfo := r.roomInfo
	if roomInfo != nil {
		return roomInfo
	}
	return nil
}

// GetRoomPlayerInfo 获得房间的某个玩家的信息
func (r *Room) GetRoomPlayerInfo(uuid string) *pb.RoomPlayerInfo {
	roomInfo := r.roomInfo
	return GetRoomPlayerInfo(roomInfo, uuid)
}

// TryJoinRoom 尝试加入房间
func (r *Room) TryJoinRoom(playerInfo *pb.PlayerInfo, joinRoomRequest *pb.GameJoinRoomRequest, extroInfo *pb.MessageExtroInfo) (*pb.RoomInfo, *pb.ErrorMessage) {
	r.rlock.Lock()
	defer r.rlock.Unlock()
	// 如果房间被标记为删除，则不能加入
	if r.roomInfo.GetDead() == true {
		return nil, nil
	}

	// 如果已经在房间中了，就直接返回断线重连后的房间信息
	for _, oneRoomPlayer := range r.roomInfo.GetPlayerInfo() {
		if oneRoomPlayer.GetUuid() == playerInfo.GetUuid() {
			return r.ResetJoinRoom(playerInfo.GetUuid()), nil
		}
	}
	// 玩家不在房间中
	// 玩家已经在其他房间中了
	if playerInfo.GetRoomId() != "" {
		return nil, nil
	}
	maxPlayerNumStr := r.GetRoomConfig("MaxPlayer")
	maxPlayerNum, err := strconv.Atoi(maxPlayerNumStr)
	if err != nil {
		return nil, nil
	}

	// 获取座位号
	// 抢座模式
	//		非断线重连 - 跳过 0-7号座位
	// 		断线重连   上面有情况判断找座位
	// 其余 有空位就坐
	seatIndex := -1
	if CheckModeOpen(pb.GameMode_GameMode_Grab) {
		if len(r.roomInfo.PlayerInfo) < 8 {
			LogError("Room TryJoinRoom grad mode length roomPlayerInfo < 8 :", len(r.roomInfo.PlayerInfo))
			return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		for index, oneRoomPlayer := range r.roomInfo.GetPlayerInfo()[8:] {
			if oneRoomPlayer.GetUuid() == "" {
				seatIndex = index + 8
				break
			}
		}
	} else {
		for index, oneRoomPlayer := range r.roomInfo.GetPlayerInfo() {
			if oneRoomPlayer.GetUuid() == "" {
				seatIndex = index
				break
			}
		}
	}

	playerGoldBean := int64(0)
	// 通过房间id加入时
	if joinRoomRequest.GetRoomUUID() != "" {
		if r.roomInfo.GetUuid() != joinRoomRequest.GetRoomUUID() {
			return nil, nil
		}
		// 到这里playerInfo.GetRoomId()肯定是为空的
		// 暂定自建房不可中途加入
		if r.roomInfo.GetRoomType() == pb.RoomType_RoomType_PlayerRoom ||
			r.roomInfo.GetRoomType() == pb.RoomType_RoomType_ClubRoom {
			// 只有第一局的ready状态才可以加入
			if r.roomInfo.GetRoomPlayNum() != 1 || r.roomInfo.GetCurRoomState() != pb.RoomState_RoomStateReady {
				return nil, GetGrpcErrorMessage(pb.ErrorCode_GameAlreadyStart, "")
			}
		}

		// 通过房间id加入时在这里判断入场金额
		enterBalanceStr := r.GetRoomConfig("EnterBalance")
		if enterBalanceStr == "" {
			LogError("Room TryJoinRoom EnterBalance config == nil", joinRoomRequest)
			return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		enterBalance, err := strconv.Atoi(enterBalanceStr)
		if err != nil {
			LogError("Room TryJoinRoom EnterBalance has err", err, joinRoomRequest, enterBalanceStr)
			return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		if int64(enterBalance) > playerInfo.GetBalance() {
			return nil, GetGrpcErrorMessage(pb.ErrorCode_LessThanEnterBalance, "")
		}

		// 如果房间有支付类型，就判断够不够支付
		if r.roomInfo.GetPayType() == pb.PayType_PayType_MasterPay {
			if r.roomInfo.GetRoomMasterUUID() == playerInfo.GetUuid() {
				if r.roomInfo.GetMasterPayNum() > playerInfo.GetBalance() {
					return nil, GetGrpcErrorMessage(pb.ErrorCode_LessThanEnterBalance, "")
				}
			}
		}
		if r.roomInfo.GetPayType() == pb.PayType_PayType_AA {
			if r.roomInfo.GetAaPayNum() > playerInfo.GetBalance() {
				return nil, GetGrpcErrorMessage(pb.ErrorCode_LessThanEnterBalance, "")
			}
		}
		if seatIndex == -1 && len(r.roomInfo.GetPlayerInfo()) >= maxPlayerNum {
			return nil, GetGrpcErrorMessage(pb.ErrorCode_RoomFull, "")
		}
		//return r.ResetJoinRoom(playerInfo.GetUuid()), nil
	}

	// 获取游戏类型
	// 如百人场，只要房间没满，他就可以重复进入该房间
	// 如对战场，他就不会重复进入之前进入的房间-而去加入下一个房间,同时根据map是遍历随机的特性，他又能不按顺序的随机去尝试。
	// 抢座模式， 可以进入之前的房间
	CateGoryStr := r.GetRoomConfig("CateGory")
	CateGoryMap := strings.Split(CateGoryStr, ",")
	for _, k := range CateGoryMap {
		CateGoryInt, err := strconv.Atoi(k)
		if err != nil {
			LogError("room TryJoinRoom atoi has error:", err)
			return nil, nil
		}
		// 是对战场，就可以不重复
		if pb.GameCateGoryType(CateGoryInt) == pb.GameCateGoryType_GameCateGoryType_Fight && !CheckModeOpen(pb.GameMode_GameMode_Grab) {
			if r.roomInfo.GetUuid() == playerInfo.GetLastRoomId() {
				return nil, nil
			}
		}
	}
	robotNum := int32(0)
	for _, oneRoomPlayer := range r.roomInfo.GetPlayerInfo() {
		if oneRoomPlayer.GetIsRobot() == true {
			robotNum++
		}
	}
	if joinRoomRequest.GetJoinRoomRobotLimit() != 0 && robotNum >= joinRoomRequest.GetJoinRoomRobotLimit() && playerInfo.GetRole() == pb.Roles_Robot {
		return nil, nil
	}

	if seatIndex == -1 && len(r.roomInfo.GetPlayerInfo()) >= maxPlayerNum {
		return nil, nil
	}
	playerInfo.GameType = r.roomInfo.GetGameType()
	playerInfo.GameScene = r.roomInfo.GetGameScene()
	playerInfo.GameServerIndex = r.roomInfo.GetGameServerIndex()
	playerInfo.RoomId = r.roomInfo.GetUuid()
	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = false
	msgErr := Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
	if msgErr != nil {
		return nil, nil
	}
	roomPlayer := &pb.RoomPlayerInfo{}
	roomPlayer.MahjongPlayerInfo = &pb.MahjongPlayerInfo{}
	roomPlayer.Uuid = playerInfo.GetUuid()
	roomPlayer.ShortId = playerInfo.GetShortId()
	roomPlayer.Account = playerInfo.GetAccount()
	roomPlayer.Name = playerInfo.GetName()
	roomPlayer.Role = playerInfo.GetRole()
	roomPlayer.Balance = playerInfo.GetBalance()
	roomPlayer.IsRobot = playerInfo.GetIsRobot()
	roomPlayer.GoldBean = playerGoldBean
	if roomPlayer.Name == "" {
		roomPlayer.Name = playerInfo.GetShortId()
	}

	// 抢座模式 - 如有在逃跑者名单中的，从名单中获取到数据
	if CheckModeOpen(pb.GameMode_GameMode_Grab) {
		var escapeeIndex = -1
		for k, v := range r.roomInfo.QPlayer {
			if v.PlayerUuid == roomPlayer.Uuid {
				escapeeIndex = k
				break
			}
		}

		if escapeeIndex != -1 {
			roomPlayer.AllWinOrLose = r.roomInfo.QPlayer[escapeeIndex].AllWinOrLose // 玩家总输赢
			//roomPlayer.BetNum = r.roomInfo.QPlayer[escapeeIndex].BetNum				// todo 手数
			roomPlayer.PunishTime = r.roomInfo.QPlayer[escapeeIndex].PunishTime     // 惩罚时间
			roomPlayer.SafeMoney = r.roomInfo.QPlayer[escapeeIndex].SafeMoney       // 桌子里的钱
			roomPlayer.AllSafeMoney = r.roomInfo.QPlayer[escapeeIndex].AllSafeMoney // 总带入桌子里的钱
		}
	}

	roomPlayer.Sex = playerInfo.GetSex()
	roomPlayer.Country = playerInfo.GetCountry()
	roomPlayer.Province = playerInfo.GetProvince()
	roomPlayer.City = playerInfo.GetCity()
	roomPlayer.HeadImgUrl = playerInfo.GetHeadImgUrl()

	if roomPlayer.Role == pb.Roles_Robot {
		roomPlayer.IsRobot = true
	} else {
		roomPlayer.IsRobot = false
	}

	roomPlayer.JoinRoomTime = time.Now().Unix()
	if r.roomInfo.GetCurRoomState() == pb.RoomState_RoomStateReady {
		roomPlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateFree
		// 不需要准备模式下，进入就直接是准备状态
		if CheckModeOpen(pb.GameMode_GameMode_NoReady) {
			roomPlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateReady
		}
	} else {
		roomPlayer.PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateWatch
	}

	// 入座
	if seatIndex == -1 {
		r.roomInfo.PlayerInfo = append(r.roomInfo.PlayerInfo, roomPlayer)
		seatIndex = len(r.roomInfo.PlayerInfo) - 1
	} else {
		r.roomInfo.PlayerInfo[seatIndex] = roomPlayer
	}

	// 空闲计数重新计算
	r.roomInfo.IdelCount = 0

	// 房卡场决定谁是点开始的人
	if GameMode == pb.GameMode_GameMode_Card && r.roomInfo.GetRoomType() != pb.RoomType_RoomType_LeagueRoom {
		firstPlayer := GetFirstJoinRoomPlayer(r.roomInfo)
		beforeStartPlayer := r.roomInfo.GetStartPlayerUUID()
		r.roomInfo.StartPlayerUUID = firstPlayer
		pushInfo := &pb.PushStartPlayerChange{}
		pushInfo.RoomId = r.roomInfo.GetUuid()
		pushInfo.BeforeStartPlayer = beforeStartPlayer
		pushInfo.AfterStartPlayer = firstPlayer
		RoomBroadcast(r.roomInfo, pushInfo)
	}

	r.Save(false)

	pushInfo := &pb.PushRoomSeatsChange{}
	pushInfo.RoomId = r.roomInfo.GetUuid()
	pushInfo.SeatIndex = int32(seatIndex)
	pushInfo.PlayerInfo = roomPlayer
	pushInfo.Reason = pb.RoomSeatsChangeReason_RoomSeatsChangeReason_Join
	RoomBroadcast(r.roomInfo, pushInfo)

	// 不需要准备模式下，有人进入就广播dotime
	if CheckModeOpen(pb.GameMode_GameMode_NoReady) == true {
		CheckPushDoTimeInReady(r.roomInfo, false)
	}

	//中途进入屏蔽一些东西
	return r.ResetJoinRoom(playerInfo.GetUuid()), nil
	//return r.roomInfo, nil
}

// TryExitRoom 尝试退出房间
func (r *Room) TryExitRoom(playerInfo *pb.PlayerInfo, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage {
	r.rlock.Lock()
	defer r.rlock.Unlock()
	// 房卡场无法中途退出
	if r.roomInfo.GetRoomType() == pb.RoomType_RoomType_ClubRoom || r.roomInfo.GetRoomType() == pb.RoomType_RoomType_PlayerRoom {
		if r.roomInfo.GetCurRoomState() != pb.RoomState_RoomStateReady ||
			(r.roomInfo.GetRoomPlayNum() >= 2 && r.roomInfo.GetRoomPlayNum() <= r.roomInfo.GetRoomAllPlayNum()) {
			return GetGrpcErrorMessage(pb.ErrorCode_NotAllowExitRoom, "")
		}
	}

	seatIndex := -1
	var allLose int64 // 输家-输钱的总额 - 正数
	for index, oneRoomPlayer := range r.roomInfo.GetPlayerInfo() {
		if oneRoomPlayer.AllWinOrLose < 0 {
			allLose -= oneRoomPlayer.AllWinOrLose
		}
		if oneRoomPlayer.GetUuid() == playerInfo.GetUuid() {
			// 当房间是对战场时，房间不能在非准备状态和玩家不能在游戏状态退出
			if !JudgeRoomIsHundred(r.roomInfo) {
				// 不是准备状态的话，只有观战玩家可以退出
				if r.roomInfo.GetCurRoomState() != pb.RoomState_RoomStateReady {
					if oneRoomPlayer.GetPlayerRoomState() != pb.PlayerRoomState_PlayerRoomStateWatch {
						return GetGrpcErrorMessage(pb.ErrorCode_NotAllowExitRoom, "")
					}
				}
				// 当房间是百人场时，玩家不能在游戏状态时退出
			} else {
				if oneRoomPlayer.GetPlayerRoomState() == pb.PlayerRoomState_PlayerRoomStatePlay {
					return GetGrpcErrorMessage(pb.ErrorCode_NotAllowExitRoom, "")
				}
			}

			seatIndex = index
			break
		}
	}
	if seatIndex == -1 {
		return GetGrpcErrorMessage(pb.ErrorCode_NotInRoom, "")
	}

	// 抢座场 - 退出有惩罚
	if CheckModeOpen(pb.GameMode_GameMode_Grab) {
		// todo
		// 退房开始计算惩罚 - 房间保存玩家
		// 玩家如果带入桌子的金额>0 - 即代表玩家上过座 - 即玩家需要计算惩罚时长
		if r.roomInfo.PlayerInfo[seatIndex].AllSafeMoney > 0 {
			r.roomInfo.QPlayer = append(r.roomInfo.QPlayer, &pb.Escapee{
				PlayerUuid:   playerInfo.Uuid,
				AllWinOrLose: r.roomInfo.PlayerInfo[seatIndex].AllWinOrLose,
				PunishTime:   r.roomInfo.PlayerInfo[seatIndex].PunishTime,
				BetNum:       1, //todo 手数用什么值???
				SafeMoney:    r.roomInfo.PlayerInfo[seatIndex].SafeMoney,
				AllSafeMoney: r.roomInfo.PlayerInfo[seatIndex].AllSafeMoney,
			})
		}

		//// 获取惩罚额度
		//penaltyQuotaStr := GetRoomConfig(r.roomInfo, "PenaltyQuota")
		//penaltyQuota, err := strconv.Atoi(penaltyQuotaStr)
		//if err != nil {
		//	LogError("Room TryExitRoom penaltyQuotaStr has err", err)
		//	return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		//}
		//
		//// 获取惩罚比例
		//penaltyRatioStr := GetRoomConfig(r.roomInfo, "PenaltyRatio")
		//penaltyRatio, err := strconv.Atoi(penaltyRatioStr)
		//if err != nil {
		//	LogError("Room TryExitRoom penaltyRatioStr has err", err)
		//	return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		//}
		//
		//// 玩家赢的钱大于额度 惩罚
		//if r.roomInfo.PlayerInfo[seatIndex].AllWinOrLose > int64(penaltyQuota) {
		//	// 惩罚值 < 玩家金额 是好多就是好多
		//	// 惩罚值 >= 玩家金额 扣光
		//	var penaltyNum = r.roomInfo.PlayerInfo[seatIndex].AllWinOrLose * int64(penaltyRatio) / 100
		//
		//	if playerInfo.Balance >= penaltyNum {
		//		realNum = penaltyNum
		//	} else {
		//		realNum = playerInfo.Balance
		//	}
		//	msgErr := AddResource(pb.RewardType_Golden, -realNum, playerInfo, true, extroInfo, true, pb.ResourceChangeReason_DaXuanQuietPenalty)
		//	if msgErr != nil {
		//		LogError("Room TryExitRoom AddResource has err", r.roomInfo.PlayerInfo[seatIndex], msgErr)
		//		return msgErr
		//	}
		//}

	}

	playerInfo.GameType = pb.GameType_None
	playerInfo.GameScene = 0
	playerInfo.GameServerIndex = ""
	playerInfo.RoomId = ""
	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = false
	msgErr := Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
	if msgErr != nil {
		return msgErr
	}

	pushInfo := &pb.PushRoomSeatsChange{}
	pushInfo.RoomId = r.roomInfo.GetUuid()
	pushInfo.SeatIndex = int32(seatIndex)
	roomPlayer := &pb.RoomPlayerInfo{}
	roomPlayer.Uuid = playerInfo.GetUuid()
	pushInfo.PlayerInfo = roomPlayer
	pushInfo.Reason = pb.RoomSeatsChangeReason_RoomSeatsChangeReason_Exit
	RoomBroadcast(r.roomInfo, pushInfo)

	defaultRoomPlayerInfo := &pb.RoomPlayerInfo{}
	r.roomInfo.PlayerInfo[seatIndex] = defaultRoomPlayerInfo
	// 房卡场退出房间,如果退出后没人了，就解散房间
	if r.roomInfo.GetRoomType() == pb.RoomType_RoomType_ClubRoom || r.roomInfo.GetRoomType() == pb.RoomType_RoomType_PlayerRoom {
		playerNum := 0
		for _, onePlayer := range r.roomInfo.GetPlayerInfo() {
			if onePlayer.GetUuid() != "" {
				playerNum++
			}
		}
		// 没有玩家了，并且是已经结束的房间
		if playerNum == 0 && r.roomInfo.GetRoomPlayNum() >= (r.roomInfo.GetRoomAllPlayNum()+1) {
			r.roomInfo.Dead = true
		}
	}

	//房间有多少人准备了，推送给所有玩家
	readyNum := 0
	for _, onePlayer := range r.roomInfo.GetPlayerInfo() {
		if onePlayer.GetPlayerRoomState() == pb.PlayerRoomState_PlayerRoomStateReady {
			readyNum++
		}
	}
	r.roomInfo.ReadyPlayerNum = int32(readyNum)
	//广播房间有多少人准备了
	pushPlayReady := &pb.RoomPlayerReadyNumMessege{
		RoomId:   r.roomInfo.Uuid,
		ReadyNum: int64(readyNum),
	}
	RoomBroadcast(r.roomInfo, pushPlayReady)

	// 不需要准备模式下，有人退出就广播dotime
	if CheckModeOpen(pb.GameMode_GameMode_NoReady) == true {
		CheckPushDoTimeInReady(r.roomInfo, false)
	}

	// 房卡场决定谁是点开始的人
	if GameMode == pb.GameMode_GameMode_Card && r.roomInfo.GetRoomType() != pb.RoomType_RoomType_LeagueRoom {
		firstPlayer := GetFirstJoinRoomPlayer(r.roomInfo)
		beforeStartPlayer := r.roomInfo.GetStartPlayerUUID()
		r.roomInfo.StartPlayerUUID = firstPlayer
		pushInfo := &pb.PushStartPlayerChange{}
		pushInfo.RoomId = r.roomInfo.GetUuid()
		pushInfo.BeforeStartPlayer = beforeStartPlayer
		pushInfo.AfterStartPlayer = firstPlayer
		RoomBroadcast(r.roomInfo, pushInfo)
	}

	r.Save(false)

	return nil
}

// Kick 踢出房间
func (r *Room) Kick(withLock bool) {
	if withLock {
		r.rlock.Lock()
		defer r.rlock.Unlock()
	}

	for index, oneRoomPlayer := range r.roomInfo.GetPlayerInfo() {
		if oneRoomPlayer.GetWaitKick() != pb.RoomSeatsChangeReason_RoomSeatsChangeReason_None &&
			oneRoomPlayer.GetPlayerRoomState() == pb.PlayerRoomState_PlayerRoomStateNone &&
			oneRoomPlayer.GetUuid() != "" {
			go HandleOtherPlayerInfo(oneRoomPlayer.GetUuid(), func(playerInfo *pb.PlayerInfo, extraInfo *pb.MessageExtroInfo) (bool, *pb.ErrorMessage) {
				playerInfo.GameType = pb.GameType_None
				playerInfo.GameScene = 0
				playerInfo.GameServerIndex = ""
				playerInfo.RoomId = ""
				return false, nil
			})
			pushInfo := &pb.PushRoomSeatsChange{}
			pushInfo.RoomId = r.roomInfo.GetUuid()
			pushInfo.SeatIndex = int32(index)
			roomPlayer := &pb.RoomPlayerInfo{}
			roomPlayer.Uuid = oneRoomPlayer.GetUuid()
			pushInfo.PlayerInfo = roomPlayer
			pushInfo.Reason = oneRoomPlayer.GetWaitKick()
			RoomBroadcast(r.roomInfo, pushInfo)

			defaultRoomPlayerInfo := &pb.RoomPlayerInfo{}
			r.roomInfo.PlayerInfo[index] = defaultRoomPlayerInfo

			r.Save(false)
		}
	}
}

//ResetJoinRoom 断线重连的回复--只给对应的一些字段
func (r *Room) ResetJoinRoom(uuid string) *pb.RoomInfo {
	reply := &pb.RoomInfo{}

	// 当是百人场 返回所有信息
	if JudgeRoomIsHundred(r.roomInfo) {
		reply = r.roomInfo
		return reply
	}

	// 当房间是百人场时，返回所有信息
	//房间id ，状态 ，下个状态 ，下个操作的玩家状态 ，房间配置信息，房间下次被驱动时间，房间游戏类型 ，场次，房间闲置数，
	//庄家索引，庄家信息，骰子，杠牌区，碰牌区，牌堆剩余长度（麻将），麻将明牌信息，最后一张牌--不变
	reply.Uuid = r.roomInfo.Uuid
	reply.CurRoomState = r.roomInfo.CurRoomState
	reply.NextRoomState = r.roomInfo.NextRoomState
	reply.DoIndex = r.roomInfo.DoIndex
	reply.Config = r.roomInfo.Config
	reply.DoTime = r.roomInfo.DoTime
	reply.GameType = r.roomInfo.GameType
	reply.GameScene = r.roomInfo.GameScene
	reply.IdelCount = r.roomInfo.IdelCount
	reply.BankerIndex = r.roomInfo.BankerIndex
	reply.BankerUuid = r.roomInfo.BankerUuid
	reply.Dice = r.roomInfo.Dice
	reply.GhMahjongInRoom = r.roomInfo.GhMahjongInRoom
	reply.LastWinnerIndex = r.roomInfo.LastWinnerIndex
	reply.JinhuaInRoom = r.roomInfo.JinhuaInRoom
	reply.RoomMasterUUID = r.roomInfo.GetRoomMasterUUID()
	reply.PayType = r.roomInfo.GetPayType()
	reply.RoomAllPlayNum = r.roomInfo.GetRoomAllPlayNum()
	reply.MasterPayNum = r.roomInfo.GetMasterPayNum()
	reply.AaPayNum = r.roomInfo.GetAaPayNum()
	reply.RoomCode = r.roomInfo.GetRoomCode()
	reply.RoomPlayNum = r.roomInfo.GetRoomPlayNum()
	reply.RoomType = r.roomInfo.GetRoomType()
	reply.ShiSanShuiCompareCurStep = r.roomInfo.ShiSanShuiCompareCurStep
	reply.StartPlayerUUID = r.roomInfo.GetStartPlayerUUID()
	reply.CrazyBullMultipleuuid = r.roomInfo.GetCrazyBullMultipleuuid()
	reply.CrazyBullMultipleuuidOdds = r.roomInfo.GetCrazyBullMultipleuuidOdds()
	reply.DoubleLinkedMultipleuuid = r.roomInfo.GetDoubleLinkedMultipleuuid()
	reply.DoubleLinkedMultipleuuidOdds = r.roomInfo.GetDoubleLinkedMultipleuuidOdds()
	reply.MahjongGameInfo = r.roomInfo.MahjongGameInfo
	reply.BenzBMWGameInfo = r.roomInfo.BenzBMWGameInfo
	reply.ClubUUID = r.roomInfo.GetClubUUID()
	reply.LeagueUUID = r.roomInfo.GetLeagueUUID()
	//扑克牌堆-->根据长度创建的全容错堆，
	PokerCardHeap := make([]*pb.Poker, 0)
	for i := 0; i < len(r.roomInfo.PokerCardHeap); i++ {
		PokerCardHeap = append(PokerCardHeap, &pb.Poker{PokerNum: pb.PokerNum_PokerNumNone,
			PokerColor: pb.PokerColor_PokerColorNone})
	}
	reply.PokerCardHeap = PokerCardHeap
	//麻将牌堆-->根据长度创建的全容错堆，
	MahjongCardHeap := make([]*pb.Mahjong, 0)
	for i := 0; i < len(r.roomInfo.GangHuaMahjongCardHeap); i++ {
		MahjongCardHeap = append(MahjongCardHeap, &pb.Mahjong{MahjongNum: pb.MahjongNum_MahjongNumNone,
			MahjongColor: pb.MahjongColor_MahjongColorNone})
	}
	reply.GangHuaMahjongCardHeap = MahjongCardHeap
	//房间玩家信息相关
	RoomPlayerInfo := make([]*pb.RoomPlayerInfo, 0)
	for _, k := range r.roomInfo.PlayerInfo {

		//该用户就是恢复重连的那个用户 就直接所有信息给他
		if uuid == k.Uuid {
			//疯狂牛牛断线重连 牌的处理(房间状态不在开牌和结算时会隐藏第五张牌)
			if r.roomInfo.GameType == pb.GameType_CrazyBull && (r.roomInfo.CurRoomState == pb.RoomState_RoomStateRushVillage || r.roomInfo.CurRoomState == pb.RoomState_RoomStateBet) && k.PlayerRoomState == pb.PlayerRoomState_PlayerRoomStatePlay {
				CrazyPlayerInfo := &pb.RoomPlayerInfo{}
				*CrazyPlayerInfo = *k
				if len(CrazyPlayerInfo.Pokers) > 0 {
					CrazyPlayerInfo.Pokers = CrazyPlayerInfo.Pokers[:4]
					RoomPlayerInfo = append(RoomPlayerInfo, CrazyPlayerInfo)
				}
			} else {
				RoomPlayerInfo = append(RoomPlayerInfo, k)
			}
			//其他用户
		} else if uuid != k.Uuid {
			//uuid ，shortId，名字 ，金额，碰牌区，杠牌区，别人打出的哪一张牌，
			//玩家房间中的状态,玩家出的扑克牌，玩家出的麻将牌，玩家输赢信息 -- 不变
			newOnePlayerInfo := &pb.RoomPlayerInfo{}
			newOnePlayerInfo.MahjongPlayerInfo = k.MahjongPlayerInfo
			newOnePlayerInfo.Uuid = k.Uuid
			newOnePlayerInfo.ShortId = k.ShortId
			newOnePlayerInfo.Account = k.Account
			newOnePlayerInfo.Name = k.Name
			newOnePlayerInfo.Balance = k.Balance
			newOnePlayerInfo.Tripletes = k.Tripletes
			newOnePlayerInfo.QuadrupletesZhi = k.QuadrupletesZhi
			newOnePlayerInfo.QuadrupletesAn = k.QuadrupletesAn
			newOnePlayerInfo.QuadrupletesBu = k.QuadrupletesBu
			newOnePlayerInfo.PlayerRoomState = k.PlayerRoomState
			newOnePlayerInfo.OutPokers = k.OutPokers
			newOnePlayerInfo.OutMahjongs = k.OutMahjongs
			newOnePlayerInfo.WinOrLose = k.WinOrLose
			newOnePlayerInfo.GhMahjongBrankerTimes = k.GhMahjongBrankerTimes
			newOnePlayerInfo.GhMahjongPoints = k.GhMahjongPoints
			newOnePlayerInfo.GhMahjongDownBankerRequest = k.GhMahjongDownBankerRequest
			newOnePlayerInfo.CrazyBullPokerType = k.CrazyBullPokerType
			newOnePlayerInfo.CrazyBullPokerOdds = k.CrazyBullPokerOdds
			newOnePlayerInfo.Point = k.GetPoint()
			newOnePlayerInfo.Sex = k.GetSex()
			newOnePlayerInfo.Country = k.GetCountry()
			newOnePlayerInfo.Province = k.GetProvince()
			newOnePlayerInfo.City = k.GetCity()
			newOnePlayerInfo.HeadImgUrl = k.GetHeadImgUrl()
			newOnePlayerInfo.ShiSanShuiPlacePoker = k.ShiSanShuiPlacePoker
			if newOnePlayerInfo.Name == "" {
				newOnePlayerInfo.Name = k.GetShortId()
			}
			newOnePlayerInfo.GoldBean = k.GetGoldBean()

			/*此处目前先返回 成 可以看到的信息，play改完后修复*/

			//别人手上的扑克牌-->全变成容错类型
			newPokers := make([]*pb.Poker, 0)
			for i := 0; i < len(k.Pokers); i++ {
				newPokers = append(newPokers, &pb.Poker{PokerNum: pb.PokerNum_PokerNumNone,
					PokerColor: pb.PokerColor_PokerColorNone})
			}
			newOnePlayerInfo.Pokers = newPokers
			//别人手上的麻将牌-->全变成容错类型
			newMahjongs := make([]*pb.Mahjong, 0)
			for i := 0; i < len(k.HandMahjongs); i++ {
				newMahjongs = append(newMahjongs, &pb.Mahjong{MahjongNum: pb.MahjongNum_MahjongNumNone,
					MahjongColor: pb.MahjongColor_MahjongColorNone})
			}
			newOnePlayerInfo.HandMahjongs = newMahjongs

			//添加给房间用户信息数组
			RoomPlayerInfo = append(RoomPlayerInfo, newOnePlayerInfo)
		}
	}

	reply.PlayerInfo = RoomPlayerInfo
	reply.GhMahjongInRoom.AfterOutCardOptes = r.roomInfo.GhMahjongInRoom.AfterOutCardOptes
	reply.GhMahjongInRoom.BeforeOutCardOpt = r.roomInfo.GhMahjongInRoom.BeforeOutCardOpt
	reply.GhMahjongInRoom.LastPushReqSelOpenCard = r.roomInfo.GhMahjongInRoom.LastPushReqSelOpenCard
	reply.GhMahjongInRoom.RemainCardNum = int32(len(r.roomInfo.GangHuaMahjongCardHeap))
	return reply
}

// PlayerUpSeat 玩家上座
func (rm *RoomManager) PlayerUpSeat(playerInfo *pb.PlayerInfo, upSeatRequest *pb.DaXuanUpSeatRequest, extraInfo *pb.MessageExtroInfo) (*pb.DaXuanUpSeatReply, *pb.ErrorMessage) {
	roomID := playerInfo.GetRoomId()
	room := rm.GetRoom(roomID)

	if room == nil {
		return nil, GetGrpcErrorMessage(pb.ErrorCode_RoomNotExist, "")
	}

	return room.TryUpSeat(playerInfo, upSeatRequest, extraInfo)
}

// TryUpSeat 尝试上座房间
func (r *Room) TryUpSeat(playerInfo *pb.PlayerInfo, realRequest *pb.DaXuanUpSeatRequest, extraInfo *pb.MessageExtroInfo) (*pb.DaXuanUpSeatReply, *pb.ErrorMessage) {
	r.rlock.Lock()
	defer r.rlock.Unlock()

	realReply := &pb.DaXuanUpSeatReply{}

	// 判断桌子长度对不对
	if len(r.roomInfo.PlayerInfo) < 8 {
		LogError("Room TryUpSeat DaXuanTable length < 8", len(r.roomInfo.PlayerInfo))
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 判断座位号是不是错的
	if realRequest.TableIndex < 0 || realRequest.TableIndex > 7 {
		LogError("Room TryUpSeat DaXuanTable has err: 7 < tableIndex < 0 ")
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_SeatIndexInvalid, "")
	}

	// 判断上桌的座位是不是有人 - 空uid || 被保座
	if r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].Uuid != "" || r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].KeepSeat {
		LogError("Room TryUpSeat DaXuanTable has err: has people!", realRequest.TableIndex, r.roomInfo.PlayerInfo[int(realRequest.TableIndex)])
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_SeatHasPeople, "")
	}

	// 获取玩家的索引
	playerIndex := GetPlayerIndex(playerInfo.Uuid, r.roomInfo)
	if playerIndex == -1 {
		LogError("Room TryUpSeat player not in room ,playerIndex = -1")
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_UserNotInRoom, "")
	}

	// 判断玩家是否已上座 如果索引0-7表示已上座
	if playerIndex >= 0 && playerIndex < 8 {
		LogError("Room TryUpSeat player already upSeat", playerIndex)
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_PlayerAlreadyInTable, "")
	}

	// 获取最小上座金额限制
	minUpSeatStr := GetRoomConfig(r.roomInfo, "MinUpSeat")
	minUpSeat, err := strconv.Atoi(minUpSeatStr)
	if err != nil {
		LogError("Room TryUpSeat minUpSeatStr has err", err)
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 获取最大上座金额限制
	maxUpSeatStr := GetRoomConfig(r.roomInfo, "MaxUpSeat")
	maxUpSeat, err := strconv.Atoi(maxUpSeatStr)
	if err != nil {
		LogError("Room TryUpSeat maxUpSeatStr has err", err)
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 上过座的玩家判断
	if r.roomInfo.PlayerInfo[playerIndex].AllSafeMoney > 0 {
		if r.roomInfo.PlayerInfo[playerIndex].SafeMoney >= int64(minUpSeat) {
			realReply.IsSuccess = true
			realReply.SafeMoney = r.roomInfo.PlayerInfo[playerIndex].SafeMoney

			// 上座 - 上座标记,台子标记,状态准备
			r.roomInfo.PlayerInfo[playerIndex].IsSeat = true
			r.roomInfo.PlayerInfo[playerIndex].PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateReady

			// 换座
			r.roomInfo.PlayerInfo[int(realRequest.TableIndex)] = r.roomInfo.PlayerInfo[playerIndex]
			r.roomInfo.PlayerInfo[playerIndex] = &pb.RoomPlayerInfo{}

			// 推送 变动信息
			pushMsg := &pb.PushTableChange{
				RoomId:        r.roomInfo.Uuid,
				TableIndex:    realRequest.TableIndex,
				PlayerUuid:    playerInfo.Uuid,
				IsUpSeat:      true,
				SafeMoney:     r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].SafeMoney,
				AllSafeMoney:  r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].AllSafeMoney,
				PlayerHeadUrl: r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].HeadImgUrl,
				PlayerShortId: r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].ShortId,
			}
			if r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].Name == "" {
				pushMsg.PlayerName = r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].ShortId
			} else {
				pushMsg.PlayerName = r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].Name
			}

			RoomBroadcast(r.roomInfo, pushMsg)

		} else {
			realReply.IsSuccess = false
			realReply.SafeMoney = r.roomInfo.PlayerInfo[playerIndex].SafeMoney
		}

		return realReply, nil
	}

	// 带入金额 小于 最小金额
	if realRequest.UpSeatMoney < int64(minUpSeat) {
		LogError("Room TryUpSeat has err:  UpSeatMoney < minUpSeatStr", realRequest.UpSeatMoney, minUpSeat)
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_BalanceTooLess, "")
	}

	// 带入金额 大于 最大金额
	if realRequest.UpSeatMoney > int64(maxUpSeat) {
		LogError("Room TryUpSeat has err:  UpSeatMoney > maxUpSeat", realRequest.UpSeatMoney, maxUpSeat)
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_BalanceTooMore, "")
	}

	// 判断钱够不够
	if r.roomInfo.PlayerInfo[playerIndex].Balance < realRequest.UpSeatMoney {
		LogError("Room TryUpSeat player balance not enough", r.roomInfo.PlayerInfo[playerIndex].Balance, realRequest.UpSeatMoney)
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_PlayerBalanceNotEnough, "")
	}

	// 扣玩家的钱 -> 同时将玩家的钱给safeMoney,allSafeMoney
	msgErr := AddResource(pb.RewardType_Golden, -realRequest.UpSeatMoney, playerInfo, false, extraInfo, true, pb.ResourceChangeReason_DaXuanUpSeat)
	if msgErr != nil {
		LogError("Room TryUpSeat AddResource has err：", msgErr)
		return realReply, msgErr
	}

	// 保存用户信息
	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = false
	msgErr = Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extraInfo)
	if msgErr != nil {
		return realReply, msgErr
	}

	// 上座 - 上座标记,台子标记,状态准备
	r.roomInfo.PlayerInfo[playerIndex].IsSeat = true
	r.roomInfo.PlayerInfo[playerIndex].PlayerRoomState = pb.PlayerRoomState_PlayerRoomStateReady

	r.roomInfo.PlayerInfo[playerIndex].Balance -= realRequest.UpSeatMoney
	r.roomInfo.PlayerInfo[playerIndex].SafeMoney += realRequest.UpSeatMoney
	r.roomInfo.PlayerInfo[playerIndex].AllSafeMoney += realRequest.UpSeatMoney

	// 换座
	r.roomInfo.PlayerInfo[int(realRequest.TableIndex)] = r.roomInfo.PlayerInfo[playerIndex]
	r.roomInfo.PlayerInfo[playerIndex] = &pb.RoomPlayerInfo{}

	realReply.IsSuccess = true
	realReply.SafeMoney = r.roomInfo.PlayerInfo[playerIndex].SafeMoney

	// 推送 变动信息
	pushMsg := &pb.PushTableChange{
		RoomId:        r.roomInfo.Uuid,
		TableIndex:    realRequest.TableIndex,
		PlayerUuid:    playerInfo.Uuid,
		IsUpSeat:      true,
		SafeMoney:     r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].SafeMoney,
		AllSafeMoney:  r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].AllSafeMoney,
		PlayerHeadUrl: r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].HeadImgUrl,
		PlayerShortId: r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].ShortId,
	}
	if r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].Name == "" {
		pushMsg.PlayerName = r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].ShortId
	} else {
		pushMsg.PlayerName = r.roomInfo.PlayerInfo[int(realRequest.TableIndex)].Name
	}
	RoomBroadcast(r.roomInfo, pushMsg)

	r.Save(false)

	return realReply, nil
}

// TopUp 玩家充值钵钵
func (rm *RoomManager) TopUp(playerInfo *pb.PlayerInfo, request *pb.DaXuanTopUpRequest, extraInfo *pb.MessageExtroInfo) (*pb.DaXuanTopUpReply, *pb.ErrorMessage) {
	roomID := playerInfo.GetRoomId()
	room := rm.GetRoom(roomID)

	if room == nil {
		return nil, GetGrpcErrorMessage(pb.ErrorCode_RoomNotExist, "")
	}

	return room.TryTopUp(playerInfo, request, extraInfo)
}

// TryTopUp 尝试玩家充值钵钵
func (r *Room) TryTopUp(playerInfo *pb.PlayerInfo, realRequest *pb.DaXuanTopUpRequest, extraInfo *pb.MessageExtroInfo) (*pb.DaXuanTopUpReply, *pb.ErrorMessage) {
	r.rlock.Lock()
	defer r.rlock.Unlock()

	realReply := &pb.DaXuanTopUpReply{}

	// 判断桌子长度对不对
	if len(r.roomInfo.PlayerInfo) < 8 {
		LogError("Room TryTopUp DaXuanTable length < 8", len(r.roomInfo.PlayerInfo))
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 获取玩家的索引
	playerIndex := GetPlayerIndex(playerInfo.Uuid, r.roomInfo)
	if playerIndex == -1 {
		LogError("Room TryTopUp player not in room ,playerIndex = -1")
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_UserNotInRoom, "")
	}

	// 获取最小上座金额限制
	minUpSeatStr := GetRoomConfig(r.roomInfo, "MinUpSeat")
	minUpSeat, err := strconv.Atoi(minUpSeatStr)
	if err != nil {
		LogError("Room TryTopUp minUpSeatStr has err", err)
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 获取最大上座金额限制
	maxUpSeatStr := GetRoomConfig(r.roomInfo, "MaxUpSeat")
	maxUpSeat, err := strconv.Atoi(maxUpSeatStr)
	if err != nil {
		LogError("Room TryTopUp maxUpSeatStr has err", err)
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	// 补充钵钵 需要玩家上过座
	// 当玩家金额 >= 最小上座金额时
	// 		玩家可以冲 (0,最大金额]
	// 当玩家金额 < 最小上座金额时
	//		玩家可以冲 [玩家金额-最小上座金额,最大金额]
	if r.roomInfo.PlayerInfo[playerIndex].AllSafeMoney > 0 {
		if r.roomInfo.PlayerInfo[playerIndex].SafeMoney >= int64(minUpSeat) {
			if realRequest.TopUpBalance <= 0 {
				LogError("Room TryTopUp top up  has err: TopUpBalance <= 0!")
				return realReply, GetGrpcErrorMessage(pb.ErrorCode_BalanceTooLess, "")
			} else if realRequest.TopUpBalance > int64(maxUpSeat) {
				LogError("Room TryTopUp top up  has err1: realRequest.TopUpBalance > maxUpSeat")
				return realReply, GetGrpcErrorMessage(pb.ErrorCode_BalanceTooMore, "")
			}
		} else {
			if realRequest.TopUpBalance < int64(minUpSeat)-r.roomInfo.PlayerInfo[playerIndex].SafeMoney {
				LogError("Room TryTopUp top up  has err: TopUpBalance < minUpSeat - SafeMoney")
				return realReply, GetGrpcErrorMessage(pb.ErrorCode_BalanceTooLess, "")
			} else if realRequest.TopUpBalance > int64(maxUpSeat) {
				LogError("Room TryTopUp top up  has err2: realRequest.TopUpBalance > maxUpSeat")
				return realReply, GetGrpcErrorMessage(pb.ErrorCode_BalanceTooMore, "")
			}
		}
	} else {
		LogError("Room TryTopUp top up  has err2: TopUp Need Seated")
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_TopUpNeedSeated, "")
	}

	// 检测玩家钱够不够
	if playerInfo.Balance < realRequest.TopUpBalance {
		LogError("Room TryTopUp top up  has err2: TopUp Need Seated")
		return realReply, GetGrpcErrorMessage(pb.ErrorCode_BalanceNotEnough, "")
	}

	// 扣玩家的钱
	msgErr := AddResource(pb.RewardType_Golden, -realRequest.TopUpBalance, playerInfo, false, extraInfo, true, pb.ResourceChangeReason_DaXuanTopUp)
	if msgErr != nil {
		LogError("Room TryTopUp AddResource has err：", msgErr)
		return realReply, msgErr
	}

	// 保存用户信息
	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = false
	msgErr = Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extraInfo)
	if msgErr != nil {
		return realReply, msgErr
	}

	// 同时将玩家的钱给safeMoney,allSafeMoney
	r.roomInfo.PlayerInfo[playerIndex].SafeMoney += realRequest.TopUpBalance
	r.roomInfo.PlayerInfo[playerIndex].AllSafeMoney += realRequest.TopUpBalance
	r.roomInfo.PlayerInfo[playerIndex].Balance -= realRequest.TopUpBalance
	realReply.IsSuccess = true
	realReply.SafeMoney = r.roomInfo.PlayerInfo[playerIndex].SafeMoney

	// 推送
	pushMsg := &pb.PushTopUp{
		RoomId:       r.roomInfo.Uuid,
		PlayerUuid:   playerInfo.Uuid,
		SafeMoney:    r.roomInfo.PlayerInfo[playerIndex].SafeMoney,
		AllSafeMoney: r.roomInfo.PlayerInfo[playerIndex].AllSafeMoney,
	}
	RoomBroadcast(r.roomInfo, pushMsg)

	r.Save(true)

	return realReply, nil
}
