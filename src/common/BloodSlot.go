package common

import (
	pb "gameServer-demo/src/grpc"
	"github.com/golang/protobuf/proto"
	"math"
	"sort"
	"strconv"
)

// 游戏接入血池控制只需要用到两个函数
// BloodGetState 控制的时候调用这个函数获取控制状态
// BloodIncrease 结算的时候调用这个函数更新血池，玩家赢钱则是负，玩家输钱则是正

// 获取血池亏盈状态（根据状态判断是否控制和控制类型）
//BloodSlotStatus_BloodSlotStatus_None  不控制 随机输赢
//BloodSlotStatus_BloodSlotStatus_Lose  平台吐分 玩家赢
//BloodSlotStatus_BloodSlotStatus_Win	平台吃粪 玩家输
func BloodGetState(gameType pb.GameType, gameScene int32) pb.BloodSlotStatus {
	gameTypeName := pb.GameType_name[int32(gameType)]
	tableKey := RedisBloodSlotTable + ":" + gameTypeName
	lockKey := MessageLockBlood + gameTypeName + ":" + strconv.Itoa(int(gameScene))
	// 分布式redis锁---👇-----
	extraInfo := &pb.MessageExtroInfo{}
	bloodMutex, err := Locker.MessageLock(lockKey, extraInfo, "BloodSlot")
	if err != nil {
		LogError("BloodGetState MessageLock has err", err)
		return pb.BloodSlotStatus_BloodSlotStatus_None
	}
	defer Locker.MessageUnlock(lockKey, extraInfo, "BloodSlot", bloodMutex)
	// 解锁 ----------👆--------

	// 获取血池信息
	info, errs := bloodGetRawObject(tableKey, gameScene)
	if errs != nil {
		//这个接口不抛出任何异常，所有异常认为血池不干预
		LogError("BloodGetRawObject has err", err)
		//解锁
		return pb.BloodSlotStatus_BloodSlotStatus_None
	}
	// 获取血池配置
	config, status, errss := bloodGetLineCfg(tableKey, gameScene)
	if errss != nil || config == nil {
		//LogError("BloodGetState the bloodGetLineCfg has err", err)
		//LogDebug("血池配置：", err)
		config = &pb.LinesConfig{}
		// 没拿到配置 status默认吃粪
		//status = pb.BloodSlotStatus_BloodSlotStatus_Eat
	}
	// 作弊值
	cheatingRatio := config.CheatRatio
	//LogDebug("血池配置：", config)
	// 强制干预次数值（用于判断未干预次数是否达到顶值）
	cfgForceNum := config.CoerceCount
	// 默认状态
	result := pb.BloodSlotStatus_BloodSlotStatus_None
	// 未干预次数
	normalNum := info.Info.FreeCount
	// 判断血池控制(如果没用血池配置则不走血池控制流程）
	if cheatingRatio != 0 {
		// 获取随机值
		random := GetRandomNum(0, 100)
		// 随机值小于作弊率或者未干预次数超出强制次数限制
		if random <= int(math.Abs(float64(cheatingRatio))) || normalNum >= cfgForceNum {
			normalNum = 0
			if cheatingRatio < 0 {
				result = pb.BloodSlotStatus_BloodSlotStatus_Lose
			}
			if cheatingRatio > 0 {
				result = pb.BloodSlotStatus_BloodSlotStatus_Win
			}
		} else {
			normalNum++
		}
	} else {
		return result
	}

	// 更新血池信息
	info.Info.FreeCount = normalNum
	info.Info.Status = status
	_ = bloodSave(info, gameType, gameScene)
	return result
}

// 获取血池配置（对外）
func BloodGetLineCfg(gameType pb.GameType, gameScene int32) (*pb.LinesConfig, pb.BloodSlotStatus, *pb.ErrorMessage) {
	gameTypeName := pb.GameType_name[int32(gameType)]
	lockKey := MessageLockBlood + gameTypeName + ":" + strconv.Itoa(int(gameScene))
	tableKey := RedisBloodSlotTable + ":" + gameTypeName
	// 分布式redis锁---👇-----
	extraInfo := &pb.MessageExtroInfo{}
	bloodMutex, err := Locker.MessageLock(lockKey, extraInfo, "BloodSlot")
	if err != nil {
		LogError("BloodGetLineCfg MessageLock has err", err)
		return nil, 0, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "BloodGetLineCfg Locker error")
	}
	defer Locker.MessageUnlock(lockKey, extraInfo, "BloodSlot", bloodMutex)
	// 解锁 ----------👆--------

	config, status, errs := bloodGetLineCfg(tableKey, gameScene)
	if errs != nil {
		LogError("bloodGetLineCfg has err", err)
		config = &pb.LinesConfig{}
		status = pb.BloodSlotStatus_BloodSlotStatus_Eat
	}
	return config, status, errs
}

// 保存血池（对外）
func BloodSave(project *pb.BloodSlotRedisStruct, gameType pb.GameType, gameScene int32) *pb.ErrorMessage {
	gameTypeName := pb.GameType_name[int32(gameType)]
	lockKey := MessageLockBlood + gameTypeName + ":" + strconv.Itoa(int(gameScene))
	// 分布式redis锁---👇-----
	extraInfo := &pb.MessageExtroInfo{}
	bloodMutex, err := Locker.MessageLock(lockKey, extraInfo, "BloodSlot")
	if err != nil {
		LogError("BloodSave MessageLock has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "BloodSave Locker error")
	}
	defer Locker.MessageUnlock(lockKey, extraInfo, "BloodSlot", bloodMutex)
	// 解锁 ----------👆--------

	errs := bloodSave(project, gameType, gameScene)
	if errs != nil {
		return errs
	}
	return nil
}

// 获取血池信息（对外）
func BloodGetRawObject(gameType pb.GameType, gameScene int32) (*pb.BloodSlotRedisStruct, *pb.ErrorMessage) {
	gameTypeName := pb.GameType_name[int32(gameType)]
	lockKey := MessageLockBlood + gameTypeName + ":" + strconv.Itoa(int(gameScene))
	tableKey := RedisBloodSlotTable + ":" + gameTypeName
	// 分布式redis锁---👇-----
	extraInfo := &pb.MessageExtroInfo{}
	bloodMutex, err := Locker.MessageLock(lockKey, extraInfo, "BloodSlot")
	if err != nil {
		LogError("BloodGetRawObject MessageLock has err", err)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "BloodGetRawObject Locker error")
	}
	defer Locker.MessageUnlock(lockKey, extraInfo, "BloodSlot", bloodMutex)
	// 解锁 ----------👆--------

	blood, errs := bloodGetRawObject(tableKey, gameScene)
	if errs != nil {
		LogError("bloodGetRawObject has err", err)
		return nil, errs
	}

	return blood, nil
}

// 更新血池值 （有扣税和暗税操作）玩家赢钱则从血池里减去，玩家输钱则总血池中新增
func BloodIncrease(value int64, gameType pb.GameType, gameScene int32) *pb.ErrorMessage {
	gameTypeName := pb.GameType_name[int32(gameType)]
	tableKey := RedisBloodSlotTable + ":" + gameTypeName
	lockKey := MessageLockBlood + gameTypeName + ":" + strconv.Itoa(int(gameScene))
	// 分布式redis锁---👇-----
	extraInfo := &pb.MessageExtroInfo{}
	bloodMutex, err := Locker.MessageLock(lockKey, extraInfo, "BloodSlot")
	if err != nil {
		LogError("BloodGetState MessageLock has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(lockKey, extraInfo, "BloodSlot", bloodMutex)
	// 解锁 ----------👆--------

	object, errs := bloodGetRawObject(tableKey, gameScene)
	if errs != nil {
		return errs
	}
	// 计算暗税
	extraTmpValue := int64(math.Abs(float64(value) / (10000 * 1000)))
	darkTax := extraTmpValue * object.Config.DarkTax * 1000
	// 更新缓存值
	object.Info.TmpValue += int64(math.Abs(float64(value % (10000 * 1000))))
	if object.Info.TmpValue >= 10000*1000 {
		// 缓存满了，加扣一次暗税并清空
		darkTax += object.Config.DarkTax * 1000
		object.Info.TmpValue = 0
	}
	// 更新血池值
	object.Info.Value += value - darkTax
	if value != 0 || darkTax != 0 {
		//记录日志（需要记录到报表中）
		LogDebug("更新血池:", object)
	}
	// 保存
	bloodSave(object, gameType, gameScene)
	return nil
}

//  只更新血池值 不做扣税和暗税计算
func BloodControl(value int64, gameType pb.GameType, gameScene int32) *pb.ErrorMessage {
	gameTypeName := pb.GameType_name[int32(gameType)]
	tableKey := RedisBloodSlotTable + ":" + gameTypeName
	lockKey := MessageLockBlood + gameTypeName + ":" + strconv.Itoa(int(gameScene))
	// 分布式redis锁---👇-----
	extraInfo := &pb.MessageExtroInfo{}
	bloodMutex, err := Locker.MessageLock(lockKey, extraInfo, "BloodSlot")
	if err != nil {
		LogError("BloodGetState MessageLock has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(lockKey, extraInfo, "BloodSlot", bloodMutex)
	// 解锁 ----------👆--------

	object, errs := bloodGetRawObject(tableKey, gameScene)
	if errs != nil {
		return errs
	}
	object.Info.Value += value
	// 保存
	bloodSave(object, gameType, gameScene)
	return nil
}

// 从redis获取对应血池数据
func bloodGetRawObject(tableKey string, gameScene int32) (*pb.BloodSlotRedisStruct, *pb.ErrorMessage) {
	// 在外层枷锁
	request := &pb.RedisMessage{
		Table: tableKey,
		Key:   strconv.Itoa(int(gameScene)),
	}
	extra := &pb.MessageExtroInfo{}
	reply := &pb.RedisMessage{}
	MsgErr := Router.Call("Redis", "HGetByte", request, reply, extra)
	if MsgErr != nil {
		LogError("GetRawObject call Redis GetByte has error：", MsgErr)
		return nil, MsgErr
	}
	//LogDebug(reply.GetValueByte())
	Blood := &pb.BloodSlotRedisStruct{}
	if reply.GetValueByte() == nil {
		// 定义初始值
		Blood.Info = &pb.BloodSlotInfo{Status: pb.BloodSlotStatus_BloodSlotStatus_Eat}
		Blood.Config = &pb.BloodSlotConfig{
			OutLines: []*pb.LinesConfig{},
			EatLines: []*pb.LinesConfig{},
		}
	} else {
		err := proto.Unmarshal(reply.GetValueByte(), Blood)
		if err != nil {
			LogError("GetRawObject  proto.Unmarshal has err:", err)
			return nil, MsgErr
		}

	}
	//LogDebug("血池数据：", Blood)
	return Blood, nil
}

// 保存血池
func bloodSave(project *pb.BloodSlotRedisStruct, gameType pb.GameType, gameScene int32) *pb.ErrorMessage {
	// 在外层枷锁
	gameTypeName := pb.GameType_name[int32(gameType)]
	tableKey := RedisBloodSlotTable + ":" + gameTypeName
	bloodRequest := &pb.RedisMessage{}
	bloodRequest.Table = tableKey
	bloodRequest.Key = strconv.Itoa(int(gameScene))
	byteValue, err := proto.Marshal(project)
	if err != nil {
		LogError("BloodSave proto.Marshal has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "BloodSlotRedisStruct marshal error")
	}
	bloodRequest.ValueByte = byteValue
	bloodReply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "HSetByte", bloodRequest, bloodReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("BloodSave Redis HSetByte has err", msgErr)
		return msgErr
	}
	return nil
}

// 获取血池配置
func bloodGetLineCfg(tableKey string, gameScene int32) (*pb.LinesConfig, pb.BloodSlotStatus, *pb.ErrorMessage) {
	// 在外层枷锁
	blood, err := bloodGetRawObject(tableKey, gameScene)
	if err != nil {
		return nil, pb.BloodSlotStatus_BloodSlotStatus_None, err
	}
	// 根据血池线 降序排序 涂粉配置和吃粪配置
	sort.Slice(blood.Config.OutLines, func(i, j int) bool {
		return blood.Config.OutLines[i].Number > blood.Config.OutLines[j].Number
	})
	sort.Slice(blood.Config.EatLines, func(i, j int) bool {
		return blood.Config.EatLines[i].Number < blood.Config.EatLines[j].Number
	})
	// 吐分状态
	if blood.Info.Status == pb.BloodSlotStatus_BloodSlotStatus_Out {
		for _, oneOutLine := range blood.Config.OutLines {
			if blood.Info.Value >= oneOutLine.Number {
				return oneOutLine, blood.Info.Status, nil
			}
		}
		for _, oneEatLine := range blood.Config.EatLines {
			if blood.Info.Value < oneEatLine.Number {
				return oneEatLine, pb.BloodSlotStatus_BloodSlotStatus_Eat, nil
			}
		}
		if len(blood.Config.OutLines) == 0 {
			return nil, blood.Info.Status, nil
		}
		//返回最后一个配置
		lastConfig := blood.Config.OutLines[len(blood.Config.OutLines)-1]
		return lastConfig, blood.Info.Status, nil
	} else if blood.Info.Status == pb.BloodSlotStatus_BloodSlotStatus_Eat {
		for _, oneEatLine := range blood.Config.EatLines {
			if blood.Info.Value <= oneEatLine.Number {
				return oneEatLine, blood.Info.Status, nil
			}
		}
		for _, oneOutLine := range blood.Config.OutLines {
			if blood.Info.Value > oneOutLine.Number {
				return oneOutLine, pb.BloodSlotStatus_BloodSlotStatus_Out, nil
			}
		}
		if len(blood.Config.OutLines) == 0 {
			return nil, blood.Info.Status, nil
		}
		//返回最后一个配置
		lastConfig := blood.Config.EatLines[len(blood.Config.EatLines)-1]
		return lastConfig, blood.Info.Status, nil
	}

	return nil, pb.BloodSlotStatus_BloodSlotStatus_Eat, nil
}

// 删除某游戏类型某场次的血池
func DeleteBloodSlot(gameType pb.GameType, gameScene int32) *pb.ErrorMessage {
	// 真实表名
	realTable := RedisBloodSlotTable + ":" + pb.GameType_name[int32(gameType)]

	// 加锁
	lockKey := MessageLockBlood + pb.GameType_name[int32(gameType)] + ":" + strconv.Itoa(int(gameScene))
	extraInfo := &pb.MessageExtroInfo{}
	roomCodeInfoMutex, err := Locker.MessageLock(lockKey, extraInfo, "BloodSlot")
	if err != nil {
		LogError("BloodSlot DeleteBloodSlot MessageLock has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	defer Locker.MessageUnlock(lockKey, extraInfo, "BloodSlot", roomCodeInfoMutex)

	// hDel
	request := &pb.RedisMessage{
		Table: realTable,
	}
	request.ValueStringArr = []string{strconv.Itoa(int(gameScene))}

	extra := &pb.MessageExtroInfo{}
	reply := &pb.RedisMessage{}
	MsgErr := Router.Call("Redis", "HDel", request, reply, extra)
	if MsgErr != nil {
		LogError("BloodSlot DeleteBloodSlot call Redis HDel has error：", MsgErr)
		return MsgErr
	}
	return nil
}
