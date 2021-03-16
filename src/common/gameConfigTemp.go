package common

import (
	pb "gameServer-demo/src/grpc"
)

// PushBobbinGameConfigTemp 推筒子配置模板
var PushBobbinGameConfigTemp map[string]*pb.GameConfig

func init() {
	// 推筒子配置模板
	pushBobbinConfigTemp()
}

// InitGameConfigTemp 预设组件要用的游戏配置模版，如果变量已存在，则不重置
// 参数要传入模版和游戏类型和游戏场次
func InitGameConfigTemp(configTemp map[string]*pb.GameConfig, gameType pb.GameType) {
	gameScenes := Configer.GetGameConfigByGameType(gameType)
	var gameScenesList []int32
	if gameScenes != nil && len(gameScenes.Map) > 0 {
		gameScenesList = make([]int32, len(gameScenes.Map))
		index := 0
		for sceneNum := range gameScenes.Map {
			gameScenesList[index] = sceneNum
			index++
		}
	} else {
		gameScenesList = make([]int32, 1)
		gameScenesList[0] = int32(1)
	}
	for _, oneScene := range gameScenesList {
		for _, config := range configTemp {
			config.GameScene = oneScene
			config.GameType = gameType
			Configer.SetGameConfig(config, false)
		}
	}
	return
}

//推筒子配置模版
func pushBobbinConfigTemp() {
	PushBobbinGameConfigTemp = make(map[string]*pb.GameConfig)
	PushBobbinGameConfigTemp["MaxPlayer"] = &pb.GameConfig{
		Name:   "MaxPlayer",
		Value:  "100",
		Remark: "推筒子的房间最大容纳的玩家数量",
	}
	PushBobbinGameConfigTemp["EnterBalance"] = &pb.GameConfig{
		Name:   "EnterBalance",
		Value:  "1000",
		Remark: "推筒子的入场限制",
	}
	PushBobbinGameConfigTemp["OutBalance"] = &pb.GameConfig{
		Name:   "OutBalance",
		Value:  "0",
		Remark: "推筒子的出场限制",
	}
	PushBobbinGameConfigTemp["DealTime"] = &pb.GameConfig{
		Name:   "DealTime",
		Value:  "3",
		Remark: "推筒子的发牌阶段时长",
	}
	PushBobbinGameConfigTemp["ReadyTime"] = &pb.GameConfig{
		Name:   "ReadyTime",
		Value:  "3",
		Remark: "推筒子的准备阶段时长",
	}
	PushBobbinGameConfigTemp["BetTime"] = &pb.GameConfig{
		Name:   "BetTime",
		Value:  "15",
		Remark: "推筒子的下注阶段时长",
	}
	PushBobbinGameConfigTemp["SettleTime"] = &pb.GameConfig{
		Name:   "SettleTime",
		Value:  "5",
		Remark: "推筒子的结算开牌阶段时长",
	}
	PushBobbinGameConfigTemp["OddsDigit0"] = &pb.GameConfig{
		Name:   "OddsDigit0",
		Value:  "1",
		Remark: "推筒子的0点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit1"] = &pb.GameConfig{
		Name:   "OddsDigit1",
		Value:  "1",
		Remark: "推筒子的1点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit2"] = &pb.GameConfig{
		Name:   "OddsDigit2",
		Value:  "1",
		Remark: "推筒子的2点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit3"] = &pb.GameConfig{
		Name:   "OddsDigit3",
		Value:  "1",
		Remark: "推筒子的3点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit4"] = &pb.GameConfig{
		Name:   "OddsDigit4",
		Value:  "1",
		Remark: "推筒子的4点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit5"] = &pb.GameConfig{
		Name:   "OddsDigit5",
		Value:  "1",
		Remark: "推筒子的5点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit6"] = &pb.GameConfig{
		Name:   "OddsDigit6",
		Value:  "1",
		Remark: "推筒子的6点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit7"] = &pb.GameConfig{
		Name:   "OddsDigit7",
		Value:  "1",
		Remark: "推筒子的7点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit8"] = &pb.GameConfig{
		Name:   "OddsDigit8",
		Value:  "1",
		Remark: "推筒子的8点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit9"] = &pb.GameConfig{
		Name:   "OddsDigit9",
		Value:  "1",
		Remark: "推筒子的9点赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit10"] = &pb.GameConfig{
		Name:   "OddsDigit10",
		Value:  "2",
		Remark: "推筒子的豹子赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit11"] = &pb.GameConfig{
		Name:   "OddsDigit11",
		Value:  "3",
		Remark: "推筒子的天杠赔率",
	}
	PushBobbinGameConfigTemp["OddsDigit12"] = &pb.GameConfig{
		Name:   "OddsDigit12",
		Value:  "4",
		Remark: "推筒子的天尊赔率",
	}
	PushBobbinGameConfigTemp["CateGory"] = &pb.GameConfig{
		Name:   "CateGory",
		Value:  "1,3",
		Remark: "推筒子的游戏类型",
	}
	PushBobbinGameConfigTemp["UserBankerMoney"] = &pb.GameConfig{
		Name:   "UserBankerMoney",
		Value:  "10000",
		Remark: "推筒子的玩家当庄所需最低金额",
	}
	PushBobbinGameConfigTemp["UserBankerRound"] = &pb.GameConfig{
		Name:   "UserBankerRound",
		Value:  "5",
		Remark: "推筒子的玩家当庄最多回合数",
	}
	PushBobbinGameConfigTemp["DefaultBankerMoney"] = &pb.GameConfig{
		Name:   "DefaultBankerMoney",
		Value:  "100000",
		Remark: "推筒子的系统当庄默认的金钱数",
	}
	PushBobbinGameConfigTemp["BankersLength"] = &pb.GameConfig{
		Name:   "BankersLength",
		Value:  "10",
		Remark: "推筒子庄家申请列表人数限制",
	}
	PushBobbinGameConfigTemp["Chips"] = &pb.GameConfig{
		Name:   "Chips",
		Value:  "1000,5000,10000,50000,100000,500000",
		Remark: "推筒子的下注的筹码值",
	}
	PushBobbinGameConfigTemp["DealMode"] = &pb.GameConfig{
		Name:   "DealMode",
		Value:  "2",
		Remark: "开牌模式：1.随机模式；2.无爆奖池模式",
	}
	PushBobbinGameConfigTemp["Commission"] = &pb.GameConfig{
		Name:   "Commission",
		Value:  "5",
		Remark: "推筒子的抽水，单位：%",
	}
}
