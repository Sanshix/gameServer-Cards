package common

import (
	pb "gameServer-demo/src/grpc"
	"time"

	"github.com/go-redsync/redsync"
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
)

// RedisLockI 分布式锁接口定义
type RedisLockI interface {
	Lock(name string) (*redsync.Mutex, error)
	Unlock(mutex *redsync.Mutex)
	MessageLock(name string, extroInfo *pb.MessageExtroInfo, componentName string) (*redsync.Mutex, error)
	MessageUnlock(name string, extroInfo *pb.MessageExtroInfo, componentName string, mutex *redsync.Mutex)
}

// LogI 日志接口定义
type LogI interface {
	Info(a ...interface{})
	Error(a ...interface{})
	Debug(a ...interface{})
}

//TimeI 时间接口定义
type TimeI interface {
	GetTimeByTimeStamp(timeStamp int64) time.Time
	GetNowWeekTimeStr() string
	IsSameDay(timeStamp1 int64, timeStamp2 int64, hour int, min int) bool
	IsSameWeek(timeStamp1 int64, timeStamp2 int64) bool
	GetTimeStr(timeStamp int64) (string, string)
	GetDate(t time.Time) int64
	ParseWeekToDate(year int, isoWeek int, weekDay time.Weekday) int64
}

// RouteI 路由接口定义
type RouteI interface {
	Call(componentName string, methodName string, request proto.Message, reply proto.Message, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage
	CallAnyReply(componentName string, methodName string, request proto.Message, extroInfo *pb.MessageExtroInfo) (proto.Message, *pb.ErrorMessage)
	CallByIp(ip string, componentName string, methodName string, request proto.Message, reply proto.Message, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage
}

// PlayerMsgSubFunc 玩家消费mq中的推送消息的函数
type PlayerMsgSubFunc func(msg []byte) error

// ReportMsgSubFunc 报表消费MQ中推送的消息回调
type ReportMsgSubFunc func(msg []byte) error

// AchievementMsgSubFunc 业绩消费MQ中推送的消息回调
type AchievementMsgSubFunc func(msg []byte) error

// CommissionMsgSubFunc 普通抽水消息回调
type CommissionMsgSubFunc func(msg []byte) error

// LeagueCommissionMsgSubFunc 大联盟抽水消息回调
type LeagueCommissionMsgSubFunc func(msg []byte) error

// PlayerStartToPlayFunc 玩家开始玩游戏回调
type PlayerStartToPlayFunc func(msg []byte) error

// MQI 消息队列接口定义
type MQI interface {
	BindUser(uuid string, subCallBack PlayerMsgSubFunc) error
	UnBindUser(uuid string) error
	SendToUser(uuid string, msg []byte) error
	SendReport(msg []byte) error
	SendAchievement(msg []byte) error
	BindReport(uuid string, subCallBack ReportMsgSubFunc) error
	UnBindReport(uuid string) error

	//发送玩家开始玩游戏通知
	SendPlayerStartToPlay(msg *pb.PlayerStartPlayGameMessage) error

	//绑定玩家开始游戏回调
	BindPlayerStartToPlay(uuid string, subCallback PlayerStartToPlayFunc) error
}

// PushI 推送器接口定义
type PushI interface {
	Push(request proto.Message, uid string)
	Broadcast(request proto.Message)
	Kick(uid string) *pb.ErrorMessage
	KickByConnId(connID string, IP string)
	SetOnline(uuid string, serverIndex string) *pb.ErrorMessage
	SetOffline(uuid string) *pb.ErrorMessage
	CheckOnline(uuid string) (bool, *pb.ErrorMessage)
	UserLoginOk(userID string, role pb.Roles, auths []pb.AuthorizationDef, clientConnID string, serverIndex string) *pb.ErrorMessage
}

// ConfigI 配置器接口定义
type ConfigI interface {
	GetGlobal(key string) *pb.GlobalConfig
	GetGlobalAll() []*pb.GlobalConfig
	//SetGlobalBatch 批量设置GlobalConfig
	SetGlobalBatch(pbs []*pb.GlobalConfig, forceUpdate bool)
	SetGlobal(data *pb.GlobalConfig, forceUpdate bool) bool

	//GetGameConfigByGameType 获取GameConfig
	GetGameConfigByGameType(gameType pb.GameType) *GameSceneMap
	//GetGameConfigByGameTypeAndScene 获取GameConfig
	GetGameConfigByGameTypeAndScene(gameType pb.GameType, gameScene int32) *GameKeyMap
	//GetGameConfig 获取GameConfig
	GetGameConfig(gameType pb.GameType, gameScene int32, name string) *pb.GameConfig
	//SetGameBatch 批量设置GameConfig
	SetGameConfigBatch(pbs []*pb.GameConfig, forceUpdate bool)
	//SetGame 设置GameConfig
	SetGameConfig(pb *pb.GameConfig, forceUpdate bool) bool
	//删除场次
	DeleteGameScene(gameType pb.GameType, gameScene int32) bool

	// 增加跑马灯配置
	AddHorseRaceLampConfig(info *pb.HorseRaceLampConfig) *pb.ErrorMessage
	// 更新跑马灯配置
	UpdateHorseRaceLampConfig(info *pb.HorseRaceLampConfig) *pb.ErrorMessage
	// 获得跑马灯配置
	GetHorseRaceLampConfig(uuid string) *pb.HorseRaceLampConfig
	// 获得所有跑马灯配置
	GetAllHorseRaceLampConfig() []*pb.HorseRaceLampConfig
	// 删除跑马灯配置
	DeleteHorseRaceLampConfig(uuid string) *pb.ErrorMessage

	// 增加机器人行为配置
	AddRobotActionConfig(info *pb.RobotActionConfig) *pb.ErrorMessage
	// 更新机器人行为配置
	UpdateRobotActionConfig(info *pb.RobotActionConfig) *pb.ErrorMessage
	// 获得机器人行为配置
	GetRobotActionConfig(uuid string) *pb.RobotActionConfig
	// 获得所有机器人行为配置
	GetAllRobotActionConfig() []*pb.RobotActionConfig

	// 增加机器人行为组配置
	AddRobotActionGroupConfig(info *pb.RobotActionGroupConfig) *pb.ErrorMessage
	// 更新机器人行为组配置
	UpdateRobotActionGroupConfig(info *pb.RobotActionGroupConfig) *pb.ErrorMessage
	// 获得机器人行为组配置
	GetRobotActionGroupConfig(uuid string) *pb.RobotActionGroupConfig
	// 获得所有机器人行为组配置
	GetAllRobotActionGroupConfig() []*pb.RobotActionGroupConfig
}

// AuthorizationI 权限验证器接口定义
type AuthorizationI interface {
	//Verify 验证接口权限
	//支持多角色判断 c.Verify("abcd.1231",pb.Roles_Agent,[1,2,3])
	Verify(authAddr string, userRole pb.Roles, userAuthMap map[pb.AuthorizationDef]pb.AuthorizationDef) bool

	//获取默认权限
	GetDefaultAuth(role pb.Roles) map[pb.AuthorizationDef]string
}

// BonusI 奖池接口定义
type BonusI interface {
	// 根据游戏类型和场次获取与比例奖池分数
	// 参数：游戏类型gameType,游戏场次gameScene,取值比例ratio(单位%),reason原因
	// 返回：分数，错误信息
	GetBonusByGameTypeAndGameScene(gameType pb.GameType, gameScene int32, ratio int, reason pb.ResourceChangeReason) (int64, *pb.BonusRecordReport, *pb.ErrorMessage)
	// 增加/减少奖池金额
	// 参数：gameType游戏类型，gameScene游戏场次，addBalance增加/减少的金额
	// 返回：金额,错误信息
	// eg:
	//     有100 +50   抽2% 	剩149	返回149,nil
	//     有50  -100  剩0		返回50,nil
	//     有50  -20   剩30		返回20,nil
	AddBonus(gameType pb.GameType, gameScene int32, incrScore int64, reason pb.ResourceChangeReason, boss bool) (int64, *pb.BonusRecordReport, *pb.ErrorMessage)
	// 只读获取奖池分数
	// 参数：游戏类型gameType,游戏场次gameScene
	// 返回：分数，错误信息
	GetBonusOnlyReady(gameType pb.GameType, gameScene int32) (int64, *pb.ErrorMessage)

	// 获取机器人的奖池分 -- 只是一个展示的分数而已,不影响真实奖池
	// 参数：gameType游戏类型，gameScene游戏场次
	// 返回：分数,错误信息
	GetRobotBonus(gameType pb.GameType, gameScene int32) (int64, *pb.ErrorMessage)
	// 增加/减少机器人奖池分数 -- 只是一个展示的分数而已,不影响真实奖池
	// 参数：gameType游戏类型，gameScene游戏场次，score分数
	// 返回：错误信息
	AddRobotBonus(gameType pb.GameType, gameScene int32, score int64) *pb.ErrorMessage

	// 增加/减少 系统奖池
	// 参数：gameType游戏类型，gameScene游戏场次，score分数
	// 返回：变动前金额,变动后金额,错误信息
	IncrBySystemBonus(gameType pb.GameType, gameScene int32, score int64) (int64, int64, *pb.ErrorMessage)

	// 只读查询 系统奖池分数
	// 参数：gameType游戏类型，gameScene游戏场次
	// 返回：分数,错误信息
	OnlyReadSystemBonus(gameType pb.GameType, gameScene int32) (int64, *pb.ErrorMessage)
}

// ReportI 报表接口
type ReportI interface {
	//发送报表数据
	SendReport(reportType pb.ReportType, reportData proto.Message) *pb.ErrorMessage
}


// Redis接口
type RedisI interface {
	StartTrans() redis.Conn
	CommitTrans(conn redis.Conn) (reply interface{}, err error)
	RollbackTrans(conn redis.Conn) error
}

type TokenI interface {
	//GenerateToken(grantType string)
	//保存第三方token
	SaveThirdPartyToken(userUUID string, tokenInfo *TokenInfo) *pb.ErrorMessage
	//验证token
	ValidateToken(accessToken string) (*TokenInfo, *pb.ErrorMessage)
}
