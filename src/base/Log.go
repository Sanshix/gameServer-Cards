package base

import (
	"fmt"
	"log"
	"os"
	"path"
	"runtime/debug"
	"sync"
	"time"

	"gameServer-demo/src/common"
)

const (
	LogBaseDir = "./log/"
)

func init() {
	common.AllComponentMap["Log"] = &Log{}
}

// Log 日志组件
type Log struct {
	common.LogI
	Base
	loggerFile  *os.File
	lCreateFile sync.Mutex
	logPath     string
	OpenDebug   bool
	Logger      *log.Logger
	HostName    string
	ToFile      bool
	LogTime     bool
}

func (self *Log) LoadComponent(config *common.OneComponentConfig, componentName string) {
	self.Base.LoadComponent(config, componentName)
	self.OpenDebug = false
	if (*self.Config)["open_debug"] == "true" {
		self.OpenDebug = true
	}
	self.ToFile = false
	if (*self.Config)["to_file"] == "true" {
		self.ToFile = true
	}
	self.LogTime = false
	if (*self.Config)["log_time"] == "true" {
		self.LogTime = true
	}
	// 获取当前目录相对应的根目录路径
	spath, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// 返回host主机名
	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	self.HostName = host
	sBasePath := path.Join(spath, LogBaseDir) + "/"
	if err := os.MkdirAll(sBasePath, os.ModePerm); err != nil {
		panic(err)
	}
	self.logPath = sBasePath
	common.LogInfo("Log LoadComponent path", self.logPath, self.HostName)
	self.createLoggerFile()
	return
}

func (self *Log) Info(a ...interface{}) {
	if self.ToFile {
		self.Logger.Println(a...)
		return
	}
	fmt.Println(a...)
}

func (self *Log) Error(a ...interface{}) {
	b := append([]interface{}{" ERROR: "}, a...)
	if self.ToFile {
		self.Logger.Println(b...)
		return
	}
	stack := string(debug.Stack())
	b = append(b, stack)
	fmt.Println(b...)
}

func (self *Log) Debug(a ...interface{}) {
	if !self.OpenDebug {
		return
	}
	if self.ToFile {
		self.Logger.Println(a...)
		return
	}
	fmt.Println(a...)
}

func (self *Log) closeFile(f *os.File) {
	go func() {
		time.Sleep(time.Second * 30)
		f.Close()
	}()
}

//记录到文件 TODO：根据日期划分文件夹 、 将error、debug和info日志分开
func (self *Log) createLoggerFile() {
	self.lCreateFile.Lock()
	defer self.lCreateFile.Unlock()

	stime := time.Now().Format("20060102150405")
	sname := ""
	for {
		//sname = self.logPath + stime + " " + common.ServerIndex + ".log"
		sname = self.logPath + common.ServerName + "_" + self.HostName + "_" + stime + ".log"
		if self.loggerFile != nil {
			self.closeFile(self.loggerFile)
		}
		f, err := os.OpenFile(sname, os.O_CREATE|os.O_EXCL|os.O_RDWR, os.ModePerm)
		if err != nil {
			continue
		}
		self.loggerFile = f
		// 日志队列
		var loggerTemp *log.Logger
		if self.LogTime {
			loggerTemp = log.New(f, "", log.LstdFlags)
		} else {
			loggerTemp = log.New(f, "", 0)
		}

		self.Logger = loggerTemp

		if self.ToFile {
			//标准输出重定向
			os.Stdout = f
			os.Stderr = f
		}

		break
	}

	//定时换
	timeNow := time.Now()
	timeNext := time.Date(timeNow.Year(), timeNow.Month(), timeNow.Day(), timeNow.Hour()+1, 0, 0, 0, time.Local)
	common.StartTimer(time.Second*time.Duration(timeNext.Unix()-timeNow.Unix()), false, func() bool {
		self.createLoggerFile()
		return false
	})
}
