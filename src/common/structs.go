package common

import (
	pb "gameServer-demo/src/grpc"
)

//GameSceneMap 游戏场次配置结构体
type GameSceneMap struct {
	GameType pb.GameType
	Map      map[int32]*GameKeyMap
}

type GameKeyMap struct {
	GameType  pb.GameType
	GameScene int32
	Map       map[string]*pb.GameConfig
}

type TokenInfo struct {
	//访问令牌
	AccessToken string
	//令牌类型 bearer/basic...
	TokenType string
	//过期时间，单位为秒
	ExpiresIn int64
	//更新令牌，用来获取下一次的访问令牌
	RefreshToken string
	//权限范围
	Scope string
	//扩展数据
	WeChatExtendedData *WeChatTokenExtendedData
}

type WeChatTokenExtendedData struct {
	OpenId  string
	UnionId string
}
