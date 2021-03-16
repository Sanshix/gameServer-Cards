// Package base 提供一些基础组件，即不管什么项目都可以用到的组件
package base

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"

	"github.com/go-redsync/redsync"
)

// Base 组件基类，提供一些组件公用的方法和属性，所有组件继承自此组件
type Base struct {
	Config        *common.OneComponentConfig
	ComponentName string
	Open          bool //是否允许外部访问
}

// LoadComponent 加载组件
func (obj *Base) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.ComponentName = componentName
	obj.Config = config
	//读取open配置，默认false
	if open, ok := (*obj.Config)["open"]; !ok {
		obj.Open = false
	} else {
		obj.Open = open == "true"
	}
	return
}

// GetConfig 获得配置
func (obj *Base) GetConfig() *common.OneComponentConfig {
	return obj.Config
}

// ComponentLock 提供分布式加锁功能，函数对common中MessageLock的封装，组件调用此函数外部将不用再传组件名称进来
//
// 参数：
//
// lockName：锁名称，常量部分请定义在common的tableDef里
//
// extroInfo：rpc消息自带的参数，里面包含了连接id，用户id，锁相关信息等
//
// 返回锁对象，如果有错误返回错误
func (obj *Base) ComponentLock(lockName string, extroInfo *pb.MessageExtroInfo) (*redsync.Mutex, error) {
	mutex, err := common.Locker.MessageLock(lockName, extroInfo, obj.ComponentName)
	return mutex, err
}

// ComponentUnlock 提供分布式解锁功能，函数对common中MessageUnlock的封装，组件调用此函数外部将不用再传组件名称进来
//
// 参数：
//
// lockName：锁名称，常量部分请定义再common的tabelDef里
//
// extroInfo：rpc消息自带的参数，里面包含了连接id，用户id，锁相关信息等
//
// mutex: 需要解锁的锁对象
func (obj *Base) ComponentUnlock(lockName string, extroInfo *pb.MessageExtroInfo, mutex *redsync.Mutex) {
	common.Locker.MessageUnlock(lockName, extroInfo, obj.ComponentName, mutex)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *Base) Start() {

}

// BeforeStart 比start更前置的方法，有的组件需要优先于其他组件的start
func (obj *Base) BeforeStart() {

}
