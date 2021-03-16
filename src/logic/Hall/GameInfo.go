package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"strings"
)

func init() {
	common.AllComponentMap["GameInfo"] = &GameInfo{}
}

type GameInfo struct {
	base.Base
	openGame []int32
}

func (self *GameInfo) LoadComponent(config *common.OneComponentConfig, componentName string) {
	self.Base.LoadComponent(config, componentName)

	Config := *self.Base.Config
	IsOpen := Config["openGame"]
	reply := make([]int32, 0)
	GameOpen := strings.Split(IsOpen, ",")
	for _, v := range GameOpen {

		relv, err := strconv.Atoi(v)
		if err != nil {
			common.LogError("GameInfo GetGameIsOpen has err", err)
			return
		}
		reply = append(reply, int32(relv))

	}
	self.openGame = reply
	common.LogDebug("开放游戏：", self.openGame)
	return
}

func (self *GameInfo) Start() {

	//self.getGameInfoSceneAndConfig(pb.GameType_CompareBull)
}

//检测游戏是否开放了---GameIsOpen?
func (self *GameInfo) GameInfo(s *pb.GameInfoRequest, extra *pb.MessageExtroInfo) (*pb.GameInfoReply, *pb.ErrorMessage) {
	reply := &pb.GameInfoReply{}
	reply.GameMap = make(map[int32]*pb.GameStatus)
	//获取开放的接口
	GameOpen := self.openGame
	//1.判断 配置没有开放接口的情况
	if GameOpen == nil {
		for _, v := range pb.GameType_value {
			if v != 0 {
				reply.GameMap[v] = &pb.GameStatus{
					IsOpen: false,
				}
			}
		}
		return reply, nil
	}
	//2.判断 配置有开放接口的情况
	//获取目前有那些游戏类型
	for _, v := range pb.GameType_value {
		for index, v1 := range GameOpen {
			//开放的接口
			if v == v1 {
				//如果没有场次信息,scenes为nil,有就填充
				config, err := self.getGameInfoSceneAndConfig(pb.GameType(v1))
				if err != nil {
					reply.GameMap[v1] = &pb.GameStatus{
						IsOpen:    true,
						Scenes:    nil,
						SortIndex: int32(index),
					}
				} else {
					reply.GameMap[v1] = &pb.GameStatus{
						IsOpen:    true,
						Scenes:    config.Scenes,
						SortIndex: int32(index),
					}
				}
				//未开放的接口,除了0
			} else if v != v1 && v != 0 && reply.GameMap[v] == nil {
				reply.GameMap[v] = &pb.GameStatus{
					IsOpen: false,
				}
			}
		}
	}
	return reply, nil
}

//获取游戏场次与配置
func (self *GameInfo) getGameInfoSceneAndConfig(gameType pb.GameType) (*pb.GameInfoGetGameSceneAndConfig, *pb.ErrorMessage) {
	reply := &pb.GameInfoGetGameSceneAndConfig{}

	sceneMap := common.Configer.GetGameConfigByGameType(gameType)
	if sceneMap == nil {
		common.LogDebug("GameInfo GetGameInfoSceneAndConfig sceneMap is empty,", gameType)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_GameSceneError, "")
	}
	reply.Scenes = make([]*pb.GameScene, 0)
	//获取场次-config
	for _, v := range sceneMap.Map {
		NewConfig := make(map[string]*pb.GameCfg)
		for _, v1 := range v.Map {
			SmallConfig := &pb.GameCfg{
				Value:  v1.Value,
				Remake: v1.Remark,
			}
			NewConfig[v1.Name] = SmallConfig
		}
		NewScene := &pb.GameScene{
			Scene:  v.GameScene,
			Config: NewConfig,
		}
		reply.Scenes = append(reply.Scenes, NewScene)
	}
	//common.LogError(reply)
	return reply, nil
}

//GetRoomCodeInfo 获取房间码的相关信息
func (g *GameInfo) GetRoomCodeInfo(request *pb.RoomCodeInfo, extra *pb.MessageExtroInfo) (*pb.RoomCodeInfo, *pb.ErrorMessage) {
	reply, msgErr := common.GetRoomCodeInfo(request.GetRoomCode())
	if reply == nil {
		reply = &pb.RoomCodeInfo{}
		msgErr = common.GetGrpcErrorMessage(pb.ErrorCode_InValidRoomCode, "")
	}
	return reply, msgErr
}

// GetBonusScore 获取奖池分数
func (g *GameInfo) GetBonusScore(request *pb.PlayerGetBonusScoreRequest, extra *pb.MessageExtroInfo) (*pb.PlayerGetBonusScoreReply, *pb.ErrorMessage) {
	reply := &pb.PlayerGetBonusScoreReply{}
	reply.GameScene = request.GameScene
	reply.GameType = request.GameType

	normal, msgErr := common.Bonuser.GetBonusOnlyReady(request.GameType, request.GameScene)
	if msgErr != nil {
		common.LogError("GameInfo GetBonusScore GetBonusOnlyReady has err:", msgErr)
		return reply, msgErr
	}

	reply.NormalBonus = normal

	return reply, nil
}
