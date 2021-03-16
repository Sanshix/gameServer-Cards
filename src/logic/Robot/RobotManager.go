package logic

import (
	"bytes"
	"errors"
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
)

func init() {
	common.AllComponentMap["RobotManager"] = &RobotManager{}
}

// RobotManager 机器人管理器组件
type RobotManager struct {
	base.Base
	prepareNum int
	openAction []pb.RobotAction
}

// LoadComponent 加载组件
func (r *RobotManager) LoadComponent(config *common.OneComponentConfig, componentName string) {
	r.Base.LoadComponent(config, componentName)
	prepareNum, err := strconv.Atoi((*r.Config)["prepare_num"])
	if err != nil {
		panic(err)
	}
	r.prepareNum = prepareNum
	openActionString := strings.Split((*r.Config)["open_action"], ",")
	// 加载当前服务机器人可操作行为配置
	for _, oneActionString := range openActionString {
		oneActionInt, err := strconv.Atoi(oneActionString)
		if err != nil {
			panic(err)
		}
		oneOpenAction := pb.RobotAction(oneActionInt)
		if oneOpenAction == pb.RobotAction_RobotAction_None {
			panic(errors.New("open_action has none"))
		}
		r.openAction = append(r.openAction, oneOpenAction)
	}

	return
}

// Start 开启组件
func (r *RobotManager) Start() {
	messageStartTime := time.Now().UnixNano() / 1e6

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
	accountInfos := queryAccountByRoleReply.GetAccountInfos()
	// 校验已有机器人信息是否初始化完毕
	for _, oneInfo := range accountInfos {
		loadPlayerRequest := &pb.LoadPlayerRequest{}
		loadPlayerRequest.Uuid = oneInfo.GetUuid()
		loadPlayerReply := &pb.LoadPlayerReply{}
		msgErr = common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
		if msgErr != nil && msgErr.GetCode() == pb.ErrorCode_DataNotFound {
			newPlayerRequest := &pb.NewPlayerRequest{}
			newPlayerRequest.Uuid = oneInfo.GetUuid()
			newPlayerRequest.ShortId = oneInfo.GetShortId()
			newPlayerRequest.RoleType = pb.Roles_Robot
			newPlayerRequest.Account = oneInfo.GetAccount()
			newPlayerReply := &pb.NewPlayerReply{}
			msgErr = common.Router.Call("PlayerInfo", "NewPlayer", newPlayerRequest, newPlayerReply, extroInfo)
			if msgErr != nil {
				common.LogError("RobotManager Start Call PlayerInfo NewPlayer has err", msgErr)
				return
			}
		}
	}
	// 判断是否应该新增机器人
	needNewRobot := r.prepareNum - len(accountInfos)
	common.LogInfo("RobotManager Start has robot num:", len(accountInfos))
	common.LogInfo("RobotManager Start prepare robot num:", r.prepareNum)
	common.LogInfo("RobotManager Start new robot num:", needNewRobot)
	if needNewRobot > 0 {
		for index := (len(accountInfos) + 1); index <= r.prepareNum; index++ {
			var buffer bytes.Buffer
			buffer.WriteString("robot-")
			buffer.WriteString("wxerct")
			buffer.WriteString(strconv.Itoa(index))
			newAccount := buffer.String()
			newPassword := "wext456gvf87"
			_, msgErr := common.Register(newAccount, newPassword, "", pb.Roles_Robot, r.ComponentName, &pb.MessageExtroInfo{}, nil)
			if msgErr != nil {
				common.LogError("RobotManager Start needNewRobot Register has err", msgErr)
				return
			}
		}
	}
	// 再重新取一次
	queryAccountByRoleRequest = &pb.QueryAccountByRoleRequest{}
	queryAccountByRoleRequest.RoleType = pb.Roles_Robot
	queryAccountByRoleReply = &pb.QueryAccountByRoleReply{}
	extroInfo = &pb.MessageExtroInfo{}
	// 初始化redis随机Id值
	msgErr = common.Router.Call("Mysql", "QueryAccountByRole", queryAccountByRoleRequest, queryAccountByRoleReply, extroInfo)
	if msgErr != nil {
		common.LogError("RobotManager Start QueryAccountByRole 2 has err", msgErr)
		return
	}
	accountInfos = queryAccountByRoleReply.GetAccountInfos()
	accountUUIDs := []string{}
	for _, oneInfo := range accountInfos {
		accountUUIDs = append(accountUUIDs, oneInfo.GetUuid())
	}
	common.LogInfo("RobotManager Start after check all robot num:", len(accountInfos))
	common.LogInfo("RobotManager Start all robot num:", len(accountUUIDs))
	//同步机器人与机器人管理信息
	r.syncRobot(accountUUIDs)
	r.syncConfig()

	// 开启工作分配的tick
	common.StartTimer(5*time.Second, false, func() bool {
		r.manage()
		return true
	})

	messageEndTime := time.Now().UnixNano() / 1e6
	costTime := messageEndTime - messageStartTime
	common.LogInfo("robot start ok", costTime)
}

// syncConfig 同步配置信息
func (r *RobotManager) syncConfig() {
	for _, oneOpenAction := range r.openAction {
		InitRobotConfigByOpenAction(oneOpenAction)
	}
}

// manage 管理操作，根据配置分配工作
func (r *RobotManager) manage() {
	common.ChangeRobotManagerInfo(func(managerInfo *pb.RobotManagerInfo) *pb.ErrorMessage {
		// 先把空闲机器人乱序
		common.RandSlice(managerInfo.FreeRobot)
		// 获得所有行为组配置
		allActionGroupConfig := common.Configer.GetAllRobotActionGroupConfig()
		for _, oneActionGroupConfig := range allActionGroupConfig {
			needRobot := int(oneActionGroupConfig.GetRobotNum())
			needIndex := -1
			// 查看组配置是否已经分配
			for index, oneActionGroup := range managerInfo.GetActionRobot() {
				if oneActionGroup.GetActionGroupUuid() == oneActionGroupConfig.GetActionGroupUuid() {
					needRobot = needRobot - len(oneActionGroup.GetRobotUuid())
					needIndex = index
					break
				}
			}
			if needRobot == 0 {
				continue
			}
			if needIndex == -1 {
				// 如果行为组不存在，就新增
				if needRobot > 0 {
					if needRobot > len(managerInfo.GetFreeRobot()) {
						needRobot = len(managerInfo.GetFreeRobot())
					}
					actionGroupInfo := &pb.RobotActionGroupInfo{}
					actionGroupInfo.ActionGroupUuid = oneActionGroupConfig.GetActionGroupUuid()
					actionGroupInfo.RobotUuid = managerInfo.FreeRobot[:needRobot]
					managerInfo.FreeRobot = append([]string{}, managerInfo.FreeRobot[needRobot:]...)
					managerInfo.ActionRobot = append(managerInfo.GetActionRobot(), actionGroupInfo)
				}
			} else {
				// 如果存在，根据新增和删除的不同情况做处理
				if needRobot > 0 {
					if needRobot > len(managerInfo.GetFreeRobot()) {
						needRobot = len(managerInfo.GetFreeRobot())
					}
					actionGroupInfo := managerInfo.ActionRobot[needIndex]
					actionGroupInfo.RobotUuid = append(actionGroupInfo.GetRobotUuid(), managerInfo.FreeRobot[:needRobot]...)
					managerInfo.FreeRobot = append([]string{}, managerInfo.FreeRobot[needRobot:]...)
				} else {
					needRobot = -needRobot
					actionGroupInfo := managerInfo.ActionRobot[needIndex]
					if needRobot > len(actionGroupInfo.GetRobotUuid()) {
						needRobot = len(actionGroupInfo.GetRobotUuid())
					}
					managerInfo.FreeRobot = append(managerInfo.GetFreeRobot(), actionGroupInfo.RobotUuid[:needRobot]...)
					actionGroupInfo.RobotUuid = append([]string{}, actionGroupInfo.RobotUuid[needRobot:]...)
				}
			}
		}
		//common.LogDebug("RobotManager:managerInfo", managerInfo)
		return nil
	})
}

// syncRobot 同步机器人于机器人管理信息
// 主要用于将新增机器人加入到空闲信息中，以及一些错误修正
func (r *RobotManager) syncRobot(accountUUIDs []string) {
	extroInfo := &pb.MessageExtroInfo{}
	// 加锁操作
	infoMutex, err := r.ComponentLock(common.MessageLockRobotManagerInfo, extroInfo)
	if err != nil {
		common.LogError("syncRobot MessageLockRobotManagerInfo has err", err)
		return
	}
	defer r.ComponentUnlock(common.MessageLockRobotManagerInfo, extroInfo, infoMutex)
	//先看redis中有没有
	redisGetRequest := &pb.RedisMessage{}
	redisGetRequest.Table = common.RedisRobotManagerInfoTable
	redisGetReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "GetByte", redisGetRequest, redisGetReply, extroInfo)
	if msgErr != nil {
		common.LogError("syncRobot Call Redis Get has err", msgErr)
		return
	}
	robotManagerInfoByte := redisGetReply.ValueByte
	robotManagerInfo := &pb.RobotManagerInfo{}
	if robotManagerInfoByte != nil {
		err = proto.Unmarshal(robotManagerInfoByte, robotManagerInfo)
		if err != nil {
			common.LogError("syncRobot RobotManagerInfo proto.Unmarshal has err", err)
			return
		}
	}
	// 先检查机器人管理信息中无效的uuid
	// 先检查空闲的
	newFreeUUIDs := []string{}
	for _, oneFreeUUID := range robotManagerInfo.GetFreeRobot() {
		accountIndex := common.IndexOf(accountUUIDs, oneFreeUUID)
		if accountIndex != -1 {
			newFreeUUIDs = append(newFreeUUIDs, oneFreeUUID)
		}
	}
	robotManagerInfo.FreeRobot = newFreeUUIDs
	// 再检查各个行为组
	for index, oneGroup := range robotManagerInfo.GetActionRobot() {
		newActionUUIDs := []string{}
		for _, oneActionUUID := range oneGroup.GetRobotUuid() {
			accountIndex := common.IndexOf(accountUUIDs, oneActionUUID)
			if accountIndex != -1 {
				newActionUUIDs = append(newActionUUIDs, oneActionUUID)
			}
		}
		robotManagerInfo.ActionRobot[index].RobotUuid = newActionUUIDs
	}

	// 再将新的加入到空闲中
	for _, oneAccountUUID := range accountUUIDs {
		group := common.GetRobotActionGroup(oneAccountUUID, robotManagerInfo)
		if group == "" {
			robotManagerInfo.FreeRobot = append(robotManagerInfo.GetFreeRobot(), oneAccountUUID)
		}
	}

	robotManagerInfoByte, err = proto.Marshal(robotManagerInfo)
	if err != nil {
		common.LogError("syncRobot RobotManagerInfo proto.Marshal has err", err)
		return
	}
	redisSetRequest := &pb.RedisMessage{}
	redisSetRequest.Table = common.RedisRobotManagerInfoTable
	redisSetRequest.ValueByte = robotManagerInfoByte
	redisSetReply := &pb.RedisMessage{}
	msgErr = common.Router.Call("Redis", "SetByte", redisSetRequest, redisSetReply, extroInfo)
	if msgErr != nil {
		common.LogError("syncRobot Redis SetByte has err", msgErr)
		return
	}
}

// GetRobotManagerConfig 获得机器人管理配置
func (r *RobotManager) GetRobotManagerConfig(request *pb.GetRobotManagerConfigRequest, extroInfo *pb.MessageExtroInfo) (*pb.GetRobotManagerConfigReply, *pb.ErrorMessage) {
	reply := &pb.GetRobotManagerConfigReply{}
	reply.PrepareNum = int32(r.prepareNum)
	reply.OpenActions = r.openAction
	return reply, nil
}
