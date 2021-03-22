# gameServer-demo

基于Golang开发的棋牌游戏服务端demo(推筒子游戏）

## 主要服务

* Hall:大厅服务
* PushBobbi:推筒子游戏服务
* Robot:机器人服务

## 环境搭建

架构使用go mod构建环境

### 基础安装

1. 将工程代码克隆在GOPATH/src目录下
2. 确保GOPATH和GOROOT的系统环境变量已经正确配置
3. 配置go mod的环境变量
* Mac或Linux
```
export GOPROXY=https://goproxy.io
export GO111MODULE=on
```
* Windows
```
set GOPROXY=https://goproxy.io
set GO111MODULE=on
```
4. 在工程目录下执行`go mod tidy`

### 运行程序 
1.  安装Go1.16
2.  设置GOROOT和GOPATH（1.6开始不需要设置GOPATH）
3.  设置环境变量
4.  根据layout.json创建layout_dev.json,其中common_config与must服务不变,其他XX_server服务里面内容都放在新的all_server服务里面
    因为本地测试需要将所有服务都同时启动，rancher上面根据环境变量运行服务
    例如：
```
    "all_server":{
         "SocketIO" : {
         "open": "true",
         "multi_line": "true"
         }, 
         "Gift": {
           "open": "true"
         }
         ...
    }
```
6.  启动配置里面各组件使用的服务（mysql、redis、MQ)

7.  进入src目录运行本地程序: 
```
go run main.go all_server 1 dev
```

### 配置文件的公共参数的意思
1.  "open":"true" 意思是允许其他组件通过grpc调用这个组件里面的函数
2.  "multi_line":"true" 意思是多线路
3.  其余参数都为组件自身需要的参数
