package common

import (
	pb "gameServer-demo/src/grpc"
)

var Logger LogI
var Locker RedisLockI
var Router RouteI
var Pusher PushI
var MQer MQI
var Timer TimeI
var Configer ConfigI
var Authorizationer AuthorizationI
var Bonuser BonusI
var Rediser RedisI
var Tokener TokenI


//组件配置名和组件配置值的映射
type OneComponentConfig map[string]string

//组件名和组件配置的映射map
type OneServerConfig map[string]OneComponentConfig

//服务器名和服务器组件map的映射
type AllServerConfig map[string]OneServerConfig

var ServerName string
var ServerConfig AllServerConfig
var GameMode pb.GameMode
var SubMode []pb.GameMode
var AllComponentMap map[string]interface{}
var ComponentMap map[string]interface{}
var ServerIndex string
var ClientInterfaceMap map[string]bool
var IsDev bool

// 初始化全局变量
func init() {
	AllComponentMap = make(map[string]interface{})
	ComponentMap = make(map[string]interface{})
	ClientInterfaceMap = make(map[string]bool)
	SubMode = make([]pb.GameMode, 0)

	ClientInterfaceMap["Register.Register"] = true
	ClientInterfaceMap["Register.RegisterByLineCode"] = true
	ClientInterfaceMap["Register.RegisterByDirectly"] = true
	ClientInterfaceMap["Register.GetCaptchaPhoto"] = true
}
