package base

import (
	"time"

	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"

	"github.com/go-redsync/redsync"
	"github.com/gomodule/redigo/redis"
)

func init() {
	common.AllComponentMap["Lock"] = &Lock{}
}

// Lock 分布式锁组件
type Lock struct {
	common.RedisLockI
	Base
	Redsync *redsync.Redsync
}

// LoadComponent 加载组件
func (obj *Lock) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)

	pools := []redsync.Pool{
		createPool((*obj.Config)["redis_host"]),
	}

	obj.Redsync = redsync.New(pools)
	return
}

// Lock 加锁，参数为锁名称，加锁失败返回错误，成功返回锁对象
func (obj *Lock) Lock(name string) (*redsync.Mutex, error) {
	mutex := obj.Redsync.NewMutex(name,
		redsync.SetExpiry(60*time.Second),
		redsync.SetTries(600),
		redsync.SetRetryDelay(100*time.Millisecond),
	)
	err := mutex.Lock()
	if err != nil {
		return nil, err
	}
	return mutex, nil
}

// Unlock 解锁，参数为锁对象
func (obj *Lock) Unlock(mutex *redsync.Mutex) {
	mutex.Unlock()
}

// MessageLock 加锁，rpc消息专用
// 参数为锁名称，rpc消息附加消息，组件名称
// 加锁失败返回错误，成功返回锁对象
func (obj *Lock) MessageLock(name string, extroInfo *pb.MessageExtroInfo, componentName string) (*redsync.Mutex, error) {
	if extroInfo.GetLocks() == nil {
		extroInfo.Locks = []*pb.MessageLock{}
	}
	for _, messageLock := range extroInfo.Locks {
		if messageLock.GetLockName() == name {
			newMessageLock := &pb.MessageLock{}
			newMessageLock.ComponentName = componentName
			newMessageLock.LockName = name
			newMessageLock.IsRealLock = false
			extroInfo.Locks = append(extroInfo.Locks, newMessageLock)
			//common.LogDebug("Lock MessageLock old lock ok", componentName, name, extroInfo.Locks)
			return nil, nil
		}
	}
	mutex, err := obj.Lock(name)
	if err != nil {
		common.LogError("Lock MessageLock new lock has err", extroInfo.Locks, componentName, name, err)
		return nil, err
	}
	newMessageLock := &pb.MessageLock{}
	newMessageLock.ComponentName = componentName
	newMessageLock.LockName = name
	newMessageLock.IsRealLock = true
	extroInfo.Locks = append(extroInfo.Locks, newMessageLock)
	//common.LogDebug("Lock MessageLock new lock ok", componentName, name, extroInfo.Locks)
	return mutex, nil
}

// MessageUnlock 解锁，rpc消息专用
// 参数为锁名称，rpc消息附加消息，组件名称
func (obj *Lock) MessageUnlock(name string, extroInfo *pb.MessageExtroInfo, componentName string, mutex *redsync.Mutex) {
	if extroInfo.GetLocks() == nil || len(extroInfo.GetLocks()) <= 0 {
		common.LogError("Lock MessageUnlock has err extroInfo.GetLocks() == nil", componentName, name)
		return
	}
	allLocks := extroInfo.GetLocks()
	lastLock := allLocks[len(allLocks)-1]
	if lastLock.GetComponentName() != componentName || lastLock.GetLockName() != name {
		common.LogError("Lock MessageUnlock has err unlock component mismatch with lock component", extroInfo.Locks, componentName, name)
		return
	}
	if lastLock.GetIsRealLock() == false {
		extroInfo.Locks = append(allLocks[:len(allLocks)-1])
		//common.LogDebug("Lock MessageUnlock old lock ok", componentName, name, extroInfo.Locks)
		return
	}
	unlockOk := mutex.Unlock()
	if unlockOk == false {
		common.LogError("Lock MessageUnlock real lock has err", extroInfo.Locks, componentName, name)
	}
	extroInfo.Locks = append(allLocks[:len(allLocks)-1])
	//common.LogDebug("Lock MessageUnlock new lock ok", componentName, name, extroInfo.Locks)
	return
}

func createPool(url string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     10,
		IdleTimeout: 180 * time.Second, // Default is 300 seconds for redis server
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", url)
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}
