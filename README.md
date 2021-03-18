# gameServer-demo

自由组件架构的游戏服务端（基于组件的可编排服务器架构）

## 编码规范

* 该架构使用golint作为代码规范检测工具，golint会以警告的形式提示编码不规范的问题，请务必解决警告后提交，主要用于规范注释问题
* redis的key统一定义在common下的tableDef文件中，mysql的表也定义在这里
* 文件名和函数名都使用驼峰命名，文件名首字母大写
* 非rpc接口请进行详细的函数注释，包括函数用途，参数说明，返回值说明
* proto文件中的协议和字段用途请进行注释
* 禁止在循环中使用"+"做字符串拼接，会对gc造成压力，内存回收会不及时，感觉像内存泄漏

## API文档

该架构使用godoc作为文档工具，Go 秉承 “注释即文档” 的理念，符合 godoc 的文档均从 Go 代码中提取并生成

### godoc使用方法

PS: Go语言 1.13 版本移除了 godoc 工具,需要手动获取：

`go get golang.org/x/tools/cmd/godoc`
1. 在命令行中输入`godoc`开启服务
2. 在浏览器中输入<http://127.0.0.1:6060/pkg/gameServer-demo/src/base> 查看base包的api和说明
3. 在浏览器中输入<http://127.0.0.1:6060/pkg/gameServer-demo/src/common> 查看common包的api和说明
4. 在浏览器中输入<http://127.0.0.1:6060/pkg/gameServer-demo/src/logic> 查看logic包的api和说明

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

### 基础工具

tools文件夹下包含了一些常用功能的脚本，酌情使用

* make_proto.sh脚本将编译proto文件，生成对应的go文件，修改proto后执行此脚本
* make.sh脚本将编译go项目源码文件，生成可执行程序到bin目录下
* upload_layout_file.sh脚本将服务器编排文件发送到静态存储
* start_test.sh脚本将在本机启动all in one模式的服务器，以便于调试所有功能
* start.sh脚本将在本机启动分布式模式的服务器，以便于调试分布式相关的功能
* stop.sh脚本将关闭本机分布式模式的服务器

### 基础发布流程

架构支持rancher搭建和linux裸机搭建两种方式，目前使用rancher搭建，以下讲述发布rancher的流程

1. 编码完毕，请先执行make.sh脚本编译源码，如果编译错误请改正
2. 编译成功后请先执行start_test.sh脚本以保证运行不会奔溃
3. 提交代码
4. upload_layout_file.sh脚本将编排文件发布到静态存储服务器
5. 等待gitlab上的自动构建脚本执行完毕
6. 去rancher上升级相关服务

如果只是服务器编排文件更改，则只执行3和4，然后重启rancher相关服务

## 架构改进

这里用来记录一些架构需要改进的地方，可能由于时间或其他原因，这些改进不能及时修改或者无法修改，记录在这里以便以后优化

1. redis和mysql组件应该做为must组件
2. 玩家信息应该像rpg一样被加载入地图，大厅也相当于地图，房间也相当于地图，这样修改玩家信息只改一处，获取玩家信息也更方便
3. 配置信息可以统一结构，减少代码量，方便理解和操作
4. 某些组件可以做成封装，而不是通过interface做成must组件

## Windows本地运行程序,proto编译

Windows编写程序与Linux编写程序基本一致，除了换行符CRLF与LF的区别会使git报错及文件权限有一些差异外暂未发现其他差异

### 运行程序 
1.  安装Go1.15及以上版本
2.  设置GOROOT和GOPATH（1.6开始不需要设置GOPATH）
3.  设置环境变量
```
GOPROXY=https://goproxy.io
GO111MODULE=on
```
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
### proto编译
1.  安装protoBuf,地址<https://github.com/protocolbuffers/protobuf/releases> 下载protoc-X.X.X-win32.zip并解压
2.  配置proto环境变量PATH里面 proto地址/bin;  
3.  获取protoc-gen-go插件，在工程目录下执行：`go get -u github.com/golang/protobuf/protoc-gen-go`（下载后的文件会被放置于$GOPATH/bin下）
4.  编译go的grpc的protoBuf文件: 
```
protoc --go_out=plugins=grpc:. message.proto  
```
这个会在生成目录生成message.pb.go文件 或者改变. 到你想他生成到什么地方的位置
### ubuntu下安装proto编译环境
1.  下载protoBuf,最新地址<https://github.com/protocolbuffers/protobuf/releases> 选择下载protobuf-all-x.xx.x.tar.gz
2.  🏠解压执行:`tar -zxvf protobuf-all-x.xx.x.tar.gz`
3.  依次执行以下命令：
    ```
    cd protobuf-all-x.xx.x
    ./configure --prefix=/usr/local/protobuf
    make
    make check
    make install
    ```
4.  配置环境变量
    ```
    export PATH=$PATH:/usr/local/protobuf/bin/
    export PKG_CONFIG_PATH=/usr/local/protobuf/lib/pkgconfig/
    source /etc/profile
    ```
5.  配置动态链接库
    ```
    sudo vim /etc/ld.so.conf
    #在文件中增加
    /usr/local/protobuf/lib
    ```
6.  执行命令:`sudo ldconfig`
7.  可能遇到问题,很有可能，执行过程中会出现以下错误提示：`./autogen.sh: 4: ./autogen.sh: autoreconf: not found`,解决办法：执行以下命令:
    ```
    sudo apt-get install autoconf
    sudo apt-get install automake
    sudo apt-get install libtool
    ```

### 配置文件的公共参数的意思
1.  "open":"true" 意思是允许其他组件通过grpc调用这个组件里面的函数
2.  "multi_line":"true" 意思是多线路
3.  其余参数都为组件自身需要的参数
