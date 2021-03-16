package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
)

func init() {
	common.AllComponentMap["Robot"] = &Robot{}
}

var robotManagerInfo *pb.RobotManagerInfo

// Roboter 机器人类
type Roboter struct {
	robotUUID         string
	roomTimer         *time.Timer
	roomTimerStopChan chan bool
}

// Robot 机器人组件
type Robot struct {
	base.Base
	handleRobotNum int
	robotUUIDs     []string
}

// LoadComponent 加载组件
func (r *Robot) LoadComponent(config *common.OneComponentConfig, componentName string) {
	r.Base.LoadComponent(config, componentName)
	handleRobotNum, err := strconv.Atoi((*r.Config)["handle_robot_num"])
	if err != nil {
		panic(err)
	}
	r.handleRobotNum = handleRobotNum
	r.robotUUIDs = []string{}
	return
}

// Start 开启组件
func (r *Robot) Start() {
	queryAccountByRoleRequest := &pb.QueryAccountByRoleRequest{}
	queryAccountByRoleRequest.RoleType = pb.Roles_Robot
	queryAccountByRoleReply := &pb.QueryAccountByRoleReply{}
	extroInfo := &pb.MessageExtroInfo{}
	//初始化redis随机Id值
	msgErr := common.Router.Call("Mysql", "QueryAccountByRole", queryAccountByRoleRequest, queryAccountByRoleReply, extroInfo)
	if msgErr != nil {
		common.LogError("RobotManager Start QueryAccountByRole has err", msgErr)
		return
	}
	serverIndex, err := strconv.Atoi(common.ServerIndex)
	if err != nil {
		panic(err)
	}
	accountInfos := queryAccountByRoleReply.GetAccountInfos()
	start := (serverIndex - 1) * r.handleRobotNum
	end := serverIndex * r.handleRobotNum
	for index, oneInfo := range accountInfos {
		if index >= start && index < end {
			r.robotUUIDs = append(r.robotUUIDs, oneInfo.GetUuid())
		}
	}
	common.StartTimer(5*time.Second, true, func() bool {
		r.syncRobotManagerInfo()
		return true
	})

	for _, oneRobotUUID := range r.robotUUIDs {
		oneRobot := new(Roboter)
		oneRobot.Init(oneRobotUUID)
	}
}

// syncRobotManagerInfo 同步机器人管理信息到内存
func (r *Robot) syncRobotManagerInfo() {
	robotManagerInfo = common.GetRobotManagerInfo()
}

// Init 机器人的初始化
func (r *Roboter) Init(UUID string) {
	r.robotUUID = UUID
	r.roomTimer, r.roomTimerStopChan = common.StartTimer(1*time.Second, false, func() bool {
		r.Drive()
		return false
	})
}

// Drive 机器人驱动
func (r *Roboter) Drive() {
	//nowTime := time.Now().Unix()
	common.StopTimer(r.roomTimer, r.roomTimerStopChan)
	nextTime, isContinue, actionConfig, actionGroupConfig := r.beforAction()
	if isContinue == true {
		isActionEnd, actionHasErr, realNextTime := r.realAction(actionConfig)
		r.afterAction(isActionEnd, actionHasErr, actionGroupConfig, actionConfig)
		nextTime = realNextTime
	}
	timerTime := nextTime
	if timerTime < 1 {
		timerTime = 1
	}
	r.roomTimer, r.roomTimerStopChan = common.StartTimer(time.Duration(timerTime)*time.Second, false, func() bool {
		r.Drive()
		return false
	})
	return
}

// resetRobotInfo 重置机器人信息
func (r *Roboter) resetRobotInfo(robotExtraInfo *pb.RobotPlayerExtroInfo) {
	robotExtraInfo.IsLaidOff = true
	robotExtraInfo.ActionGroupUuid = ""
	robotExtraInfo.CurActionUuid = ""
}

// beforAction 执行action之前的检查
// 返回下次驱动的时间，是否继续执行action
func (r *Roboter) beforAction() (int64, bool, *pb.RobotActionConfig, *pb.RobotActionGroupConfig) {
	uuid := r.robotUUID
	extroInfo := &pb.MessageExtroInfo{}
	componentName := "Roboter"
	playerMutex, err := common.Locker.MessageLock(common.MessageLockPlayer+uuid, extroInfo, componentName)
	if err != nil {
		common.LogError("Roboter beforAction MessageLock has err", err)
		return 5, false, nil, nil
	}
	defer common.Locker.MessageUnlock(common.MessageLockPlayer+uuid, extroInfo, componentName, playerMutex)

	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		common.LogError("Roboter beforAction LoadPlayer has err", msgErr)
		return 5, false, nil, nil
	}
	playerInfo := loadPlayerReply.GetPlayerInfo()

	actionGroup := common.GetRobotActionGroup(r.robotUUID, robotManagerInfo)
	if playerInfo.GetRobotExtroInfo() == nil {
		playerInfo.RobotExtroInfo = &pb.RobotPlayerExtroInfo{}
	}
	robotExtraInfo := playerInfo.GetRobotExtroInfo()
	// 机器人没有配置工作
	if actionGroup == "free" || actionGroup == "" {
		// 机器人没有正在进行的工作，则重置机器人，以便下次循环
		if robotExtraInfo.GetActionGroupUuid() == "" {
			r.resetRobotInfo(robotExtraInfo)
			// 顺便让机器人下线一次，容错
			common.Pusher.SetOffline(r.robotUUID)
			savePlayerRequest := &pb.SavePlayerRequest{}
			savePlayerRequest.PlayerInfo = playerInfo
			savePlayerRequest.ForceSave = false
			msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
			if msgErr != nil {
				return 5, false, nil, nil
			}
			return 10, false, nil, nil
		}
		// 机器人有正在进行的工作，则让机器人下岗先
		robotExtraInfo.IsLaidOff = true
	} else {
		// 机器人有配置工作
		// 机器人没有正在运行的工作，就让机器人上岗
		if robotExtraInfo.GetActionGroupUuid() == "" {
			robotExtraInfo.IsLaidOff = false
			// 错误的配置，让机器人重置
			actionGroupConfig := common.Configer.GetRobotActionGroupConfig(actionGroup)
			if actionGroupConfig == nil {
				common.LogError("actionGroup can not find config", actionGroup)
				r.resetRobotInfo(robotExtraInfo)
				savePlayerRequest := &pb.SavePlayerRequest{}
				savePlayerRequest.PlayerInfo = playerInfo
				savePlayerRequest.ForceSave = false
				msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
				if msgErr != nil {
					return 5, false, nil, nil
				}
				return 5, false, nil, nil
			}
			robotExtraInfo.ActionGroupUuid = actionGroup
		} else {
			// 机器人有正在运行的工作
			// 但和配置的工作不同,则标记为下岗先
			if actionGroup != robotExtraInfo.GetActionGroupUuid() {
				robotExtraInfo.IsLaidOff = true
			} else {
				// 和配置的工作相同
			}
		}
	}
	// 校验机器人当前行为组配置
	actionGroupConfig := common.Configer.GetRobotActionGroupConfig(robotExtraInfo.GetActionGroupUuid())
	if actionGroupConfig == nil {
		common.LogError("robotExtraInfo action group can not find config", robotExtraInfo.GetActionGroupUuid())
		r.resetRobotInfo(robotExtraInfo)
		savePlayerRequest := &pb.SavePlayerRequest{}
		savePlayerRequest.PlayerInfo = playerInfo
		savePlayerRequest.ForceSave = false
		msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
		if msgErr != nil {
			return 5, false, nil, nil
		}
		return 5, false, nil, nil
	}
	// 校验机器人行为
	if robotExtraInfo.GetCurActionUuid() == "" {
		robotExtraInfo.CurActionUuid = actionGroupConfig.GetActionConfigsUuid()[0]
	}
	// 校验机器人行为配置
	actionConfig := common.Configer.GetRobotActionConfig(robotExtraInfo.GetCurActionUuid())
	if actionConfig == nil {
		common.LogError("robotExtraInfo action can not find config", robotExtraInfo.GetCurActionUuid())
		r.resetRobotInfo(robotExtraInfo)
		savePlayerRequest := &pb.SavePlayerRequest{}
		savePlayerRequest.PlayerInfo = playerInfo
		savePlayerRequest.ForceSave = false
		msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
		if msgErr != nil {
			return 5, false, nil, nil
		}
		return 5, false, nil, nil
	}
	// 检测行为有效性
	if _, ok := ActionList[actionConfig.GetActionType()]; !ok {
		common.LogError("action type err", actionConfig.GetActionType())
		r.resetRobotInfo(robotExtraInfo)
		savePlayerRequest := &pb.SavePlayerRequest{}
		savePlayerRequest.PlayerInfo = playerInfo
		savePlayerRequest.ForceSave = false
		msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
		if msgErr != nil {
			return 5, false, nil, nil
		}
		return 5, false, nil, nil
	}

	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = false
	msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
	if msgErr != nil {
		return 5, false, nil, nil
	}
	return 5, true, actionConfig, actionGroupConfig
}

// realAction 真正的行动
// 返回是否行动结束，行动是否有错误，下次执行的时间
func (r *Roboter) realAction(actionConfig *pb.RobotActionConfig) (bool, bool, int64) {
	uuid := r.robotUUID
	extroInfo := &pb.MessageExtroInfo{}
	componentName := "Roboter"
	playerMutex, err := common.Locker.MessageLock(common.MessageLockPlayer+uuid, extroInfo, componentName)
	if err != nil {
		common.LogError("Roboter realAction MessageLock has err", err)
		return false, false, 5
	}
	defer common.Locker.MessageUnlock(common.MessageLockPlayer+uuid, extroInfo, componentName, playerMutex)

	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		common.LogError("Roboter realAction LoadPlayer has err", msgErr)
		return false, false, 5
	}
	playerInfo := loadPlayerReply.GetPlayerInfo()
	if playerInfo.GetRobotExtroInfo() == nil {
		playerInfo.RobotExtroInfo = &pb.RobotPlayerExtroInfo{}
	}
	robotExtraInfo := playerInfo.GetRobotExtroInfo()
	// 获得机器人所在的房间信息
	var roomInfo *pb.RoomInfo
	if playerInfo.GetRoomId() != "" && playerInfo.GetGameType() != pb.GameType_None {
		roomInfoRequest := &pb.RedisMessage{}
		roomInfoRequest.Table = common.GetRoomRedisName(playerInfo.GetGameType(), playerInfo.GetGameServerIndex())
		roomInfoRequest.Key = playerInfo.GetRoomId()
		roomInfoReply := &pb.RedisMessage{}
		msgErr := common.Router.Call("Redis", "HGetByte", roomInfoRequest, roomInfoReply, &pb.MessageExtroInfo{})
		if msgErr != nil {
			common.LogError("Roboter realAction get room info has err", playerInfo.GetRoomId(), playerInfo.GetGameType(), playerInfo.GetGameServerIndex())
			return false, false, 5
		}
		// 房间不存在，重置机器人
		if roomInfoReply.GetValueByte() == nil {
			playerInfo.GameType = pb.GameType_None
			playerInfo.GameScene = 0
			playerInfo.GameServerIndex = ""
			playerInfo.RoomId = ""
			r.resetRobotInfo(robotExtraInfo)

			savePlayerRequest := &pb.SavePlayerRequest{}
			savePlayerRequest.PlayerInfo = playerInfo
			savePlayerRequest.ForceSave = false
			msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
			if msgErr != nil {
				common.LogError("Roboter realAction SavePlayer has err", playerInfo.GetUuid())
				return false, false, 5
			}
			common.LogError("Roboter realAction room not find", playerInfo.GetRoomId(), playerInfo.GetGameType(), playerInfo.GetGameServerIndex())
			return false, false, 5
		}

		roomInfo = &pb.RoomInfo{}
		err := proto.Unmarshal(roomInfoReply.GetValueByte(), roomInfo)
		if err != nil {
			playerInfo.GameType = pb.GameType_None
			playerInfo.GameScene = 0
			playerInfo.GameServerIndex = ""
			playerInfo.RoomId = ""
			r.resetRobotInfo(robotExtraInfo)

			savePlayerRequest := &pb.SavePlayerRequest{}
			savePlayerRequest.PlayerInfo = playerInfo
			savePlayerRequest.ForceSave = false
			msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
			if msgErr != nil {
				common.LogError("Roboter realAction SavePlayer2 has err", playerInfo.GetUuid())
				return false, false, 5
			}
			common.LogError("Roboter realAction Unmarshal room has err", playerInfo.GetRoomId(), playerInfo.GetGameType(), playerInfo.GetGameServerIndex())
			return false, false, 5
		}
	}
	extroInfo.UserId = r.robotUUID
	// 一切正常，执行行为
	isActionEnd, hasErr, nextTime := ActionList[actionConfig.GetActionType()].Action(playerInfo, roomInfo, actionConfig, extroInfo)
	return isActionEnd, hasErr, nextTime
}

// afterAction 执行action之后的处理
// 返回下次驱动的时间，是否继续执行action
func (r *Roboter) afterAction(isActionEnd bool, actionHasErr bool, actionGroupConfig *pb.RobotActionGroupConfig, actionConfig *pb.RobotActionConfig) {
	uuid := r.robotUUID
	extroInfo := &pb.MessageExtroInfo{}
	componentName := "Roboter"
	playerMutex, err := common.Locker.MessageLock(common.MessageLockPlayer+uuid, extroInfo, componentName)
	if err != nil {
		common.LogError("Roboter afterAction MessageLock has err", err)
		return
	}
	defer common.Locker.MessageUnlock(common.MessageLockPlayer+uuid, extroInfo, componentName, playerMutex)

	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		common.LogError("Roboter afterAction LoadPlayer has err", msgErr)
		return
	}
	playerInfo := loadPlayerReply.GetPlayerInfo()
	if playerInfo.GetRobotExtroInfo() == nil {
		playerInfo.RobotExtroInfo = &pb.RobotPlayerExtroInfo{}
	}
	robotExtraInfo := playerInfo.GetRobotExtroInfo()

	if actionHasErr == true {
		robotExtraInfo.ErrNum = robotExtraInfo.ErrNum + 1
	} else {
		robotExtraInfo.ErrNum = 0
	}
	if robotExtraInfo.GetErrNum() > 10 {
		robotExtraInfo.IsLaidOff = true
	}
	// 如果行为结束，就切换行为
	if isActionEnd == true {
		actionConfigIndex := common.IndexOf(actionGroupConfig.GetActionConfigsUuid(), actionConfig.GetActionUuid())
		// 当前行为不存在了，放生错误
		if actionConfigIndex == -1 {
			r.resetRobotInfo(robotExtraInfo)
			common.LogError("Roboter afterAction cur action config not found", actionConfig.GetActionUuid())
			savePlayerRequest := &pb.SavePlayerRequest{}
			savePlayerRequest.PlayerInfo = playerInfo
			savePlayerRequest.ForceSave = false
			msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
			if msgErr != nil {
				return
			}
			return
		}
		if actionConfigIndex >= (len(actionGroupConfig.GetActionConfigsUuid()) - 1) {
			robotExtraInfo.CurActionUuid = actionGroupConfig.GetActionConfigsUuid()[0]
		} else {
			robotExtraInfo.CurActionUuid = actionGroupConfig.GetActionConfigsUuid()[actionConfigIndex+1]
		}
	}
	savePlayerRequest := &pb.SavePlayerRequest{}
	savePlayerRequest.PlayerInfo = playerInfo
	savePlayerRequest.ForceSave = false
	msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, &pb.EmptyMessage{}, extroInfo)
	if msgErr != nil {
		common.LogError("Roboter realAction SavePlayer2 has err", playerInfo.GetUuid())
		return
	}
}
