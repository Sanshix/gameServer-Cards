package main

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"syscall"
	"time"

	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"gameServer-demo/src/logic"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	var configBytes []byte

	//读取服务编排配置
	common.IsDev = false
	configFileName := "./layout_dev.json"
	if len(os.Args) > 3 && os.Args[3] == "dev" {
		configFileName = "./layout_dev.json"
	}
	common.IsDev = true
	file, err := os.Open(configFileName)
	if err != nil {
		common.LogError("open layout_dev.json has err", err)
		return
	}
	defer file.Close()
	configBytes, err = ioutil.ReadAll(file)
	file.Close()
	err = json.Unmarshal(configBytes, &common.ServerConfig)
	if err != nil {
		common.LogError(" json.Unmarshal has err", err)
		return
	}
	common.LogInfo("common.ServerConfig", common.ServerConfig)
	//读取服务器环境变量，加载组件
	serverName := os.Getenv("SERVER_NAME")
	if len(os.Args) > 1 {
		serverName = os.Args[1]
	}
	common.ServerName = serverName
	common.LogInfo("common.ServerName:", common.ServerName)
	common.ServerIndex = "0"
	serverIndex := os.Getenv("SERVER_INDEX")
	if len(os.Args) > 2 {
		serverIndex = os.Args[2]
	}
	if serverIndex != "" {
		common.ServerIndex = serverIndex
	}
	common.LogInfo("common.ServerIndex:", common.ServerIndex)
	baseServerConfig := common.ServerConfig["base_config"]
	if baseServerConfig == nil {
		common.LogError("baseServerConfig == nil")
		return
	}
	gameModeStr := baseServerConfig["GameMode"]["mode"]
	gameModeInt, err := strconv.Atoi(gameModeStr)
	if err != nil {
		common.LogError("gameModeStr atoi has err", err)
		return
	}
	gameMode := pb.GameMode(gameModeInt)
	if gameMode == pb.GameMode_GameMode_None {
		common.LogError("gameMode is none")
		return
	}
	common.GameMode = gameMode
	common.LogInfo("common.GameMode:", common.GameMode)

	curServerConfig := common.ServerConfig[serverName]
	if curServerConfig == nil {
		common.LogError("curServerConfig == nil")
		return
	}
	commonServerConfig := common.ServerConfig["common_config"]
	if commonServerConfig == nil {
		common.LogError("commonServerConfig == nil")
		return
	}
	mustServerConfig := common.ServerConfig["must"]
	if mustServerConfig == nil {
		common.LogError("mustServerConfig == nil")
		return
	}
	//引入组件初始化
	base.Init()
	logic.Init()

	//先加载基础组件
	for componentName, componentConfig := range mustServerConfig {
		oneComponentConfig := common.OneComponentConfig{}
		//先用公共配置的值来填充
		commonComponentConfig := commonServerConfig[componentName]
		if commonComponentConfig != nil {
			for oneFieldName, oneFieldValue := range commonComponentConfig {
				oneComponentConfig[oneFieldName] = oneFieldValue
			}
		}
		//然后使用基础组件配置填充
		for oneFieldName, oneFieldValue := range componentConfig {
			oneComponentConfig[oneFieldName] = oneFieldValue
		}
		//检测是否多线程服务并确认组件名
		realComponentName := componentName
		if oneComponentConfig["multi_line"] == "true" {
			realComponentName = componentName + serverIndex
		}
		//过滤重复组件
		if common.ComponentMap[realComponentName] != nil {
			continue
		}
		//检测组件是否初始化
		if common.AllComponentMap[componentName] == nil {
			common.LogError("init component err, componentName == nil", componentName)
			return
		}
		//最后使用各自进程的组件配置填充
		curComponentConfig := curServerConfig[componentName]
		if curComponentConfig != nil {
			for oneFieldName, oneFieldValue := range curComponentConfig {
				oneComponentConfig[oneFieldName] = oneFieldValue
			}
		}
		//调用组件加载LoadComponent函数
		methodArgs := []reflect.Value{reflect.ValueOf(&oneComponentConfig), reflect.ValueOf(realComponentName)}
		reflect.ValueOf(common.AllComponentMap[componentName]).MethodByName("LoadComponent").Call(methodArgs)
		common.ComponentMap[realComponentName] = common.AllComponentMap[componentName]
	}
	//开始加载进程独有组件
	for componentName, componentConfig := range curServerConfig {
		oneComponentConfig := common.OneComponentConfig{}
		//先用公共配置的值来填充
		commonComponentConfig := commonServerConfig[componentName]
		if commonComponentConfig != nil {
			for oneFieldName, oneFieldValue := range commonComponentConfig {
				oneComponentConfig[oneFieldName] = oneFieldValue
			}
		}
		//然后使用组件自己的配置填充
		for oneFieldName, oneFieldValue := range componentConfig {
			oneComponentConfig[oneFieldName] = oneFieldValue
		}
		realComponentName := componentName
		if oneComponentConfig["multi_line"] == "true" {
			realComponentName = componentName + serverIndex
		}
		//过滤重复
		if common.ComponentMap[realComponentName] != nil {
			continue
		}
		//检测组件是否初始化
		if common.AllComponentMap[componentName] == nil {
			common.LogError("init component err, componentName == nil", componentName)
			return
		}
		//调用组件加载LoadComponent函数
		methodArgs := []reflect.Value{reflect.ValueOf(&oneComponentConfig), reflect.ValueOf(realComponentName)}
		reflect.ValueOf(common.AllComponentMap[componentName]).MethodByName("LoadComponent").Call(methodArgs)
		common.ComponentMap[realComponentName] = common.AllComponentMap[componentName]
	}

	//开启分布式锁组件
	lockComponentInterface := common.ComponentMap["Lock"]
	if lockComponentInterface != nil {
		lockComponent, ok := lockComponentInterface.(*base.Lock)
		if !ok {
			common.LogError(" lockComponentInterface not lockComponent ")
			return
		}
		common.Locker = lockComponent
	}
	//开启消息路由组件
	routeComponentInterface := common.ComponentMap["Route"]
	if routeComponentInterface != nil {
		routeComponent, ok := routeComponentInterface.(*base.Route)
		if !ok {
			common.LogError(" routeComponentInterface not routeComponent ")
			return
		}
		common.Router = routeComponent
	}
	//开启推送组件
	pushComponentInterface := common.ComponentMap["Push"]
	if pushComponentInterface != nil {
		pushComponent, ok := pushComponentInterface.(*base.Push)
		if !ok {
			common.LogError(" pushComponentInterface not pushComponent ")
			return
		}
		common.Pusher = pushComponent
	}
	//开启MQ组件
	mqComponentInterface := common.ComponentMap["MQ"]
	if mqComponentInterface != nil {
		mqComponent, ok := mqComponentInterface.(*base.MQ)
		if !ok {
			common.LogError(" mqComponentInterface not mqComponent ")
			return
		}
		common.MQer = mqComponent
	}
	//开启时间组件
	timeComponentInterface := common.ComponentMap["Time"]
	if timeComponentInterface != nil {
		timeComponent, ok := timeComponentInterface.(*base.Time)
		if !ok {
			common.LogError(" timeComponentInterface not timeComponent ")
			return
		}
		common.Timer = timeComponent
	}

	//global config component
	configComponentInterface := common.ComponentMap["Config"]
	if configComponentInterface != nil {
		configComponent, ok := configComponentInterface.(*base.Config)
		if !ok {
			common.LogError(" configComponentInterface not configComponent ")
			return
		}
		common.Configer = configComponent
	}

	//global config component
	authComponentInterface := common.ComponentMap["Authorization"]
	if authComponentInterface != nil {
		authComponent, ok := authComponentInterface.(*base.Authorization)
		if !ok {
			common.LogError(" configComponentInterface not configComponent ")
			return
		}
		common.Authorizationer = authComponent
	}

	//开启日志服务
	logComponentInterface := common.ComponentMap["Log"]
	if logComponentInterface != nil {
		logComponent, ok := logComponentInterface.(*base.Log)
		if !ok {
			common.LogError(" logComponentInterface not logComponent ")
			return
		}
		common.Logger = logComponent
	}

	redisComponentInterface := common.ComponentMap["Redis"]
	if redisComponentInterface != nil {
		redisComponent, ok := redisComponentInterface.(*base.Redis)
		if !ok {
			common.LogError(" redisComponentInterface not pushComponent ")
			return
		}
		common.Rediser = redisComponent
	}

	tokenComponentInterface := common.ComponentMap["Token"]
	if tokenComponentInterface != nil {
		tokenComponent, ok := tokenComponentInterface.(*base.Token)
		if !ok {
			common.LogError(" tokenComponentInterface not pushComponent ")
			return
		}
		common.Tokener = tokenComponent
	}
	// 所有组件加载完毕，初始化也完毕的情况下，调用BeforeStart方法
	// 有些组件需要更前置的执行
	for _, component := range common.ComponentMap {
		reflect.ValueOf(component).MethodByName("BeforeStart").Call([]reflect.Value{})
	}

	// 所有组件加载完毕，初始化也完毕的情况下，调用Start方法
	for _, component := range common.ComponentMap {
		reflect.ValueOf(component).MethodByName("Start").Call([]reflect.Value{})
	}

	//所有完毕之后，才开放,开启服务发现与注册服务
	findComponentInterface := common.ComponentMap["Find"]
	if findComponentInterface != nil {
		findComponent, ok := findComponentInterface.(*base.Find)
		if !ok {
			common.LogError(" findComponentInterface not findComponent ")
			return
		}
		findComponent.RegisterComponent()
	}

	stime := time.Now().Format("2006-01-02 15:04:05")
	common.LogInfo("server start ok", common.ServerName, common.ServerIndex, stime)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		syscall.SIGINT,
		syscall.SIGILL,
		syscall.SIGFPE,
		syscall.SIGSEGV,
		syscall.SIGTERM,
		syscall.SIGABRT)
	<-signalChan
	common.LogInfo("do some close operate")
	// 清除连接信息
	socketInterface := common.ComponentMap["SocketIO"+serverIndex]
	if socketInterface != nil {
		// 断言验证
		socketComponent, ok := socketInterface.(*base.SocketIO)
		if !ok {
			common.LogError(" socketInterface not socketComponent ")
			return
		}
		socketComponent.Clear()
	}
	common.LogInfo("server end")

}
