package base

import (
	"errors"
	"math/rand"
	"net"
	"reflect"
	"strings"
	"time"

	"gameServer-demo/src/common"

	"github.com/gomodule/redigo/redis"
)

func init() {
	common.AllComponentMap["Find"] = &Find{}
}

// Find服务发现
type Find struct {
	Base
	RedisPool redis.Pool
	serverIp  string
	grpcPort  string
}

func (self *Find) LoadComponent(config *common.OneComponentConfig, componentName string) {
	self.Base.LoadComponent(config, componentName)

	self.RedisPool = redis.Pool{
		MaxIdle:     16,
		IdleTimeout: 180 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", (*self.Config)["redis_host"])
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	conn, err := net.Dial("udp", "www.google.com.hk:80")
	defer conn.Close()
	if err != nil {
		panic(err)
	}
	self.serverIp = strings.Split(conn.LocalAddr().String(), ":")[0]
	self.grpcPort = (*self.Config)["grpc_port"]
	common.LogInfo("FindComponent serverIp : ", self.serverIp, self.grpcPort)
	return
}

// 定时循环注册本地服务并保存到redis
func (self *Find) RegisterComponent() {
	// 循环写入是为了服务挂掉后redis中数据会过期，这样其他服务就不会再调用挂掉的服务
	common.StartTimer(500*time.Millisecond, false, func() bool {
		conn := self.RedisPool.Get()
		defer conn.Close()
		//curServerConfig := common.ServerConfig[common.ServerName]
		conn.Send("MULTI")
		for componentName, component := range common.ComponentMap {
			//判断是否是开放组件
			methodArgs := []reflect.Value{}
			rst := reflect.ValueOf(component).MethodByName("GetConfig").Call(methodArgs)
			// 转成interface再断言成接口类型
			config := rst[0].Interface().(*common.OneComponentConfig)
			if open, ok := (*config)["open"]; ok && open == "true" {
				componentInfo := componentName + "_" + self.serverIp + ":" + self.grpcPort
				conn.Send("SETEX", componentInfo, 1, "1")
			}
		}
		_, err := conn.Do("EXEC")
		if err != nil {
			common.LogError("RegisterComponent op redis has err", err)
		}
		return true
	})
}

// 查询组件并返回组件ip
func (self *Find) FindComponent(componentName string) (string, error) {
	conn := self.RedisPool.Get()
	defer conn.Close()
	rst, err := redis.Strings(conn.Do("KEYS", componentName+"*"))
	if err != nil {
		return "", err
	}
	if len(rst) <= 0 {
		return "", errors.New("Find FindComponent not find this component:" + componentName)
	}
	// 如果多线路则随机路线
	n := rand.Intn(len(rst))
	ip := strings.Split(rst[n], "_")[1]
	return ip, nil
}

// 获取所有组件服务的链接地址
func (self *Find) FindAllComponent(componentName string) (map[string]bool, error) {
	conn := self.RedisPool.Get()
	defer conn.Close()
	ips := make(map[string]bool)
	rst, err := redis.Strings(conn.Do("KEYS", componentName+"*"))
	if err != nil {
		return ips, err
	}
	if len(rst) <= 0 {
		return ips, nil
	}

	for _, ip := range rst {
		realIp := strings.Split(ip, "_")[1]
		ips[realIp] = true
	}

	return ips, nil
}
