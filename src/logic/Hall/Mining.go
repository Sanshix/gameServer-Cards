package logic

import (
	"encoding/json"
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"github.com/golang/protobuf/proto"
	"math"
	"strconv"
	"time"
)

func init() {
	common.AllComponentMap["Mining"] = &Mining{}
}

// 挖矿组件
type Mining struct {
	base.Base
}

// LoadComponent 加载组件
func (m *Mining) LoadComponent(config *common.OneComponentConfig, componentName string) {
	m.Base.LoadComponent(config, componentName)
	return
}

// Start 开启组件
func (m *Mining) Start() {
	initGlobalConfigNameArr := []string{
		"MiningDefault",
		"MiningMaxLevel",
		"MiningLevelOfNew",
	}
	err := common.InitGlobleConfigTemp(initGlobalConfigNameArr)
	if err != nil {
		panic(err)
	}
}

//获取挖矿进度
func (m *Mining) GetMining(request *pb.GetMiningProgressRequest, extroInfo *pb.MessageExtroInfo) (*pb.GetMiningProgresReply, *pb.ErrorMessage) {
	//获取玩家信息
	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = extroInfo.GetUserId()
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		common.LogError("GetMining  Get PlayerInfo has err ", msgErr)
		return nil, msgErr
	}
	// 对工人查岗
	reply, ok := m._getMiningObject(loadPlayerReply.PlayerInfo.Uuid)
	if ok == false {
		//新来的矿工，老规矩先捡肥皂
		reply, msgErr = m._getMiningDefault(loadPlayerReply.PlayerInfo)
		if msgErr != nil {
			return nil, msgErr
		}
	}
	return reply, nil
}

//领取挖矿奖励
func (m *Mining) ReceiveMining(request *pb.GetMiningRequest, extroInfo *pb.MessageExtroInfo) (*pb.GetMiningReply, *pb.ErrorMessage) {
	//获取玩家信息
	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = extroInfo.GetUserId()
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extroInfo)
	if msgErr != nil {
		common.LogError("GetMining  Get PlayerInfo has err ", msgErr)
		return nil, msgErr
	}
	// 检查工作
	miningInfo, ok := m._getMiningObject(loadPlayerReply.PlayerInfo.Uuid)
	newTime := time.Now().Unix()
	// 干得不好不发工资
	if ok == false || newTime < miningInfo.EndTime {
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_MiningNotCondition, "")
	}
	// 安排新工作
	newReply, msgErr := m._getMiningDefault(loadPlayerReply.PlayerInfo)
	if msgErr != nil {
		return nil, msgErr
	}
	// 发工资
	msgErr = common.AddResource(pb.RewardType_Diamond, miningInfo.Reward, loadPlayerReply.PlayerInfo, true, extroInfo, true, pb.ResourceChangeReason_MiningReward)
	if msgErr != nil {
		common.LogError("ReceiveMining AddResource wagerNum has err", loadPlayerReply.PlayerInfo.GetUuid, miningInfo.Reward, msgErr)
		return nil, msgErr
	}

	reply := &pb.GetMiningReply{
		IsSuccess: true,
		Rewards:   miningInfo.Reward,
		NewMining: newReply,
	}
	return reply, nil
}

// 从redis获取挖矿进度
func (m *Mining) _getMiningObject(Uuid string) (*pb.GetMiningProgresReply, bool) {
	request := &pb.RedisMessage{
		Table: common.RedisMiningProgressTable,
		Key:   Uuid,
	}
	extra := &pb.MessageExtroInfo{}
	reply := &pb.RedisMessage{}
	MsgErr := common.Router.Call("Redis", "HGetByte", request, reply, extra)
	if MsgErr != nil {
		common.LogError("GetRawObject call Redis GetByte has error：", MsgErr)
		return nil, false
	}
	common.LogDebug("_getMiningObject", reply.GetValueByte())
	if reply.GetValueByte() == nil {
		return nil, false
	}
	info := &pb.GetMiningProgresReply{}
	err := proto.Unmarshal(reply.GetValueByte(), info)
	if err != nil {
		common.LogError("_getMiningObject  proto.Unmarshal has err:", err)
		return nil, false
	}
	return info, true
}

// 从redis更新挖矿进度
func (m *Mining) _setMiningObject(Uuid string, info *pb.GetMiningProgresReply) *pb.ErrorMessage {
	infoData, err := json.Marshal(info)
	if err != nil {
		common.LogError("mashal token data err", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	request := &pb.RedisMessage{
		Table:     common.RedisMiningProgressTable,
		Key:       Uuid,
		ValueByte: infoData,
	}
	extra := &pb.MessageExtroInfo{}
	reply := &pb.RedisMessage{}
	MsgErr := common.Router.Call("Redis", "HSetByte", request, reply, extra)
	if MsgErr != nil {
		common.LogError("GetRawObject call Redis GetByte has error：", MsgErr)
		return MsgErr
	}
	return nil
}

// 分配挖矿工作
func (m *Mining) _getMiningDefault(PlayerInfo *pb.PlayerInfo) (*pb.GetMiningProgresReply, *pb.ErrorMessage) {
	// 获取最大等级
	MiningMaxLevel := common.Configer.GetGlobal("MiningMaxLevel").Value
	MiningMaxLevelInt, err1 := strconv.Atoi(MiningMaxLevel)
	// 获取最低挖矿工资
	MiningDefault := common.Configer.GetGlobal("MiningDefault").Value
	MiningDefaultInt, err2 := strconv.Atoi(MiningDefault)
	// 获取升职后的加薪
	MiningLevelOfNew := common.Configer.GetGlobal("MiningLevelOfNew").Value
	MiningLevelOfNewInt, err3 := strconv.Atoi(MiningLevelOfNew)
	if err1 != nil || err2 != nil || err3 != nil {
		common.LogError("_getMiningDefault data Atoi has err", err1, err2, err3)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	orderReply := &pb.MysqlOrderInfo{}
	// 获取当前总充值金额
	msgErr := common.Router.Call("Mysql", "GetOrderGoldCountByUUID", PlayerInfo, orderReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		common.LogError("OrderInfo getIsFirstChargeGiftAuth Call Mysql has err", msgErr)
		return nil, msgErr
	}
	// 获取矿工当前职位
	nowLevel := int(math.Floor(float64(orderReply.ProductNum / 10)))
	if nowLevel >= MiningMaxLevelInt {
		nowLevel = MiningMaxLevelInt
	}
	// 当前薪资
	nowDiamond := ((nowLevel - 1) * MiningLevelOfNewInt) + MiningDefaultInt

	// 18个小时发一次工资 ~~ 1080分钟
	nowTime := time.Now().Unix()
	endTime := nowTime + (1080 * 60)

	reply := &pb.GetMiningProgresReply{
		Level:   int64(nowLevel),
		EndTime: endTime,
		Reward:  int64(nowDiamond),
	}
	// 更新redis记录
	msgErr = m._setMiningObject(PlayerInfo.Uuid, reply)
	if msgErr != nil {
		return nil, msgErr
	}

	return reply, nil
}
