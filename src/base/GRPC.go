package base

import (
	"context"
	"errors"
	"net"
	"reflect"
	"runtime/debug"
	"sync"
	"time"

	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/keepalive"

	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

var (
	// 下面的 grpc client 参数设置,来自官方示范代码
	// https://github.com/grpc/grpc-go/blob/master/examples/features/keepalive/client/main.go
	kacp = keepalive.ClientParameters{
		Time:                10 * time.Second, // send pings every 10 seconds if there is no activity
		Timeout:             time.Second,      // wait 1 second for ping ack before considering the connection dead
		PermitWithoutStream: true,             // send pings even without active streams
	}

	// 下面的 grpc client 参数设置,来自官方示范代码
	// https://github.com/grpc/grpc-go/blob/master/examples/features/keepalive/server/main.go
	kaep = keepalive.EnforcementPolicy{
		MinTime:             5 * time.Second, // If a client pings more than once every 5 seconds, terminate the connection
		PermitWithoutStream: true,            // Allow pings even when there are no active streams
	}

	kasp = keepalive.ServerParameters{
		MaxConnectionIdle:     15 * time.Second, // If a client is idle for 15 seconds, send a GOAWAY
		MaxConnectionAge:      30 * time.Second, // If any connection is alive for more than 30 seconds, send a GOAWAY
		MaxConnectionAgeGrace: 5 * time.Second,  // Allow 5 seconds for pending RPCs to complete before forcibly closing connections
		Time:                  5 * time.Second,  // Ping the client if it is idle for 5 seconds to ensure the connection is still active
		Timeout:               1 * time.Second,  // Wait 1 second for the ping ack before assuming the connection is dead
	}
)

func init() {
	common.AllComponentMap["GRPC"] = &GRPC{}
}

type ComponentConn struct {
	Conn          *grpc.ClientConn
	ComponentName string
	Ip            string
}

type ComponentConns map[string]ComponentConn

// GRPC 用于转发信息与外部服务交互
type GRPC struct {
	Base
	ConnPool          map[string]ComponentConns
	lConnPool         sync.Mutex
	refreshConnsTimer *time.Ticker
}

func (self *GRPC) HandleMessage(ctx context.Context, in *pb.HandleMessageRequest) (*pb.HandleMessageReply, error) {
	defer func() {
		if err := recover(); err != nil {
			common.LogError("GRPC HandleMessage has err", in, err)
			debug.PrintStack()
		}
	}()
	grpcReply := &pb.HandleMessageReply{}
	componentName := in.GetComponentName()
	methodName := in.GetMethodName()
	inMessage := in.GetMessageContent()
	extroInfo := in.GetExtroInfo()
	realInMessage, err := ptypes.Empty(inMessage)
	if err != nil {
		common.LogError("GRPC HandleMessage ptypes.Empty has err", err)
		grpcReply.ErrMessage = common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		return grpcReply, nil
	}
	err = ptypes.UnmarshalAny(inMessage, realInMessage)
	if err != nil {
		common.LogError("GRPC HandleMessage ptypes.UnmarshalAny has err", err)
		grpcReply.ErrMessage = common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		return grpcReply, nil
	}
	if common.ComponentMap[componentName] != nil {
		methodArgs := []reflect.Value{reflect.ValueOf(realInMessage), reflect.ValueOf(extroInfo)}
		rst := reflect.ValueOf(common.ComponentMap[componentName]).MethodByName(methodName).Call(methodArgs)
		var reply proto.Message
		if rst[0].Interface() != nil {
			reply = rst[0].Interface().(proto.Message)
		} else {
			reply = nil
		}
		var msgErr *pb.ErrorMessage
		if rst[1].Interface() != nil {
			msgErr = rst[1].Interface().(*pb.ErrorMessage)
		} else {
			msgErr = nil
		}
		if msgErr != nil {
			//common.LogError("GRPC HandleMessage Call has err", msgErr)
			grpcReply.ErrMessage = msgErr
			return grpcReply, nil
		}

		replyAny, err := ptypes.MarshalAny(reply)
		if err != nil {
			common.LogError("GRPC HandleMessage MarshalAny(reply) has err", err)
			grpcReply.ErrMessage = common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
			return grpcReply, nil
		}
		grpcReply.MessageContent = replyAny
		return grpcReply, nil
	}

	common.LogError("GRPC HandleMessage componentName err", componentName)
	grpcReply.ErrMessage = common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	return grpcReply, nil
}

func (self *GRPC) LoadComponent(config *common.OneComponentConfig, componentName string) {
	self.Base.LoadComponent(config, componentName)
	self.ConnPool = make(map[string]ComponentConns)
	go func() {
		listen, err := net.Listen("tcp", (*self.Config)["listen_url"])
		if err != nil {
			panic(err)
		}
		//实现gRPC Server
		s := grpc.NewServer(grpc.KeepaliveEnforcementPolicy(kaep), grpc.KeepaliveParams(kasp))
		//注册helloServer为客户端提供服务
		pb.RegisterGRPCComponentServer(s, self) //内部调用了s.RegisterServer()
		common.LogInfo("GRPCComponent Listen on", (*self.Config)["listen_url"])

		s.Serve(listen)
	}()

	common.StartTimer(5*time.Second, false, func() bool {
		self.RefreshConn()
		return true
	})

	return
}

func (self *GRPC) SendMessage(componentName string, methodName string, request proto.Message, extroInfo *pb.MessageExtroInfo) (proto.Message, *pb.ErrorMessage) {
	defer func() {
		if err := recover(); err != nil {
			common.LogError("GRPC SendMessage has err", err)
			debug.PrintStack()
		}
	}()
	componentConn, err := self.NewConn(componentName)
	var reply proto.Message
	if err != nil {
		common.LogError("GRPC SendMessage NewConn has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	grpcContent, err := ptypes.MarshalAny(request)
	if err != nil {
		common.LogError("GRPC SendMessage MarshalAny(request) has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//初始化客户端
	c := pb.NewGRPCComponentClient(componentConn.Conn)
	//调用方法
	reqBody := &pb.HandleMessageRequest{}
	reqBody.ComponentName = componentName
	reqBody.MethodName = methodName
	reqBody.MessageContent = grpcContent
	reqBody.ExtroInfo = extroInfo
	ctx1, cel := context.WithTimeout(context.Background(), time.Second*10)
	defer cel()
	r, err := c.HandleMessage(ctx1, reqBody)
	if err != nil {
		common.LogError("GRPC SendMessage Call HandleMessage has err", componentName, methodName, request, extroInfo, err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	if r.GetErrMessage() != nil {
		//common.LogError("GRPC SendMessage Call HandleMessage GetErrMessage has err", componentName, methodName, request, extroInfo, r.GetErrMessage())
		return reply, r.GetErrMessage()
	}
	grpcReply := r.GetMessageContent()
	realGrpcReply, err := ptypes.Empty(grpcReply)
	if err != nil {
		common.LogError("GRPC SendMessage ptypes.Empty has err", err)
		return grpcReply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	err = ptypes.UnmarshalAny(grpcReply, realGrpcReply)
	if err != nil {
		common.LogError("GRPC SendMessage UnmarshalAny(grpcReply, reply) has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return realGrpcReply, nil
}

func (self *GRPC) SendMessageByIp(ip string, componentName string, methodName string, request proto.Message, extroInfo *pb.MessageExtroInfo) (proto.Message, *pb.ErrorMessage) {
	defer func() {
		if err := recover(); err != nil {
			common.LogError("GRPC SendMessageByIp has err", err)
			debug.PrintStack()
		}
	}()
	componentConn, err := self.NewConnByIp(ip, componentName)
	var reply proto.Message
	if err != nil {
		common.LogError("GRPC SendMessageByIp NewConn has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	grpcContent, err := ptypes.MarshalAny(request)
	if err != nil {
		common.LogError("GRPC SendMessageByIp MarshalAny(request) has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//初始化客户端
	c := pb.NewGRPCComponentClient(componentConn.Conn)
	//调用方法
	reqBody := &pb.HandleMessageRequest{}
	reqBody.ComponentName = componentName
	reqBody.MethodName = methodName
	reqBody.MessageContent = grpcContent
	reqBody.ExtroInfo = extroInfo
	ctx1, cel := context.WithTimeout(context.Background(), time.Second*10)
	defer cel()
	r, err := c.HandleMessage(ctx1, reqBody)
	if err != nil {
		common.LogError("GRPC SendMessageByIp Call HandleMessage has err", componentName, methodName, request, extroInfo, err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	if r.GetErrMessage() != nil {
		//common.LogError("GRPC SendMessageByIp Call HandleMessage GetErrMessage has err", componentName, methodName, request, extroInfo, err)
		return reply, r.GetErrMessage()
	}
	grpcReply := r.GetMessageContent()
	realGrpcReply, err := ptypes.Empty(grpcReply)
	if err != nil {
		common.LogError("GRPC SendMessageByIp ptypes.Empty has err", err)
		return grpcReply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	err = ptypes.UnmarshalAny(grpcReply, realGrpcReply)
	if err != nil {
		common.LogError("GRPC SendMessageByIp UnmarshalAny(grpcReply, reply) has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return realGrpcReply, nil
}

func (self *GRPC) DeleteConn(conn *ComponentConn) {
	self.lConnPool.Lock()
	defer self.lConnPool.Unlock()

	self.deleteConnNoLock(conn)
}

func (self *GRPC) deleteConnNoLock(conn *ComponentConn) {
	if self.ConnPool[conn.ComponentName] == nil {
		return
	}

	conn.Conn.Close()
	delete(self.ConnPool[conn.ComponentName], conn.Ip)
}

func (self *GRPC) NewConn(componentName string) (*ComponentConn, error) {
	self.lConnPool.Lock()
	defer self.lConnPool.Unlock()

	if self.ConnPool[componentName] == nil || len(self.ConnPool[componentName]) <= 0 {
		findComponentInterface := common.ComponentMap["Find"]
		findComponent, _ := findComponentInterface.(*Find)
		ip, err := findComponent.FindComponent(componentName)
		if err != nil {
			return nil, err
		}

		conn, err := grpc.Dial(ip, grpc.WithBlock(), grpc.WithInsecure(), grpc.WithKeepaliveParams(kacp))
		if err != nil {
			return nil, err
		}

		if self.ConnPool[componentName] == nil {
			self.ConnPool[componentName] = make(ComponentConns)
		}

		componentConn := ComponentConn{}
		componentConn.Conn = conn
		componentConn.ComponentName = componentName
		componentConn.Ip = ip
		self.ConnPool[componentName][ip] = componentConn
		return &componentConn, nil
	}

	conns := self.ConnPool[componentName]
	for _, conn := range conns {
		if conn.Conn.GetState() != connectivity.Ready {
			//common.LogError("grpc client status: ", conn.Conn.GetState().String())
			continue
		}

		return &conn, nil
	}
	return nil, errors.New("GRPC NewConn no conn:" + componentName)
}

func (self *GRPC) NewConnByIp(ip string, componentName string) (*ComponentConn, error) {
	self.lConnPool.Lock()
	defer self.lConnPool.Unlock()

	findComponentInterface := common.ComponentMap["Find"]
	findComponent, _ := findComponentInterface.(*Find)
	ips, err := findComponent.FindAllComponent(componentName)
	if err != nil {
		return nil, err
	}
	if _, ok := ips[ip]; !ok {
		return nil, errors.New("GRPC NewConnByIp " + componentName + " not has this ip:" + ip)
	}

	if self.ConnPool[componentName] == nil {
		self.ConnPool[componentName] = make(ComponentConns)
	}
	if _, ok := self.ConnPool[componentName][ip]; !ok {
		conn, err := grpc.Dial(ip, grpc.WithBlock(), grpc.WithInsecure(), grpc.WithKeepaliveParams(kacp))
		if err != nil {
			return nil, err
		}

		componentConn := ComponentConn{}
		componentConn.Conn = conn
		componentConn.ComponentName = componentName
		componentConn.Ip = ip
		self.ConnPool[componentName][ip] = componentConn
		return &componentConn, nil
	}

	conn := self.ConnPool[componentName][ip]
	return &conn, nil
}

func (self *GRPC) RefreshConn() {
	self.lConnPool.Lock()
	defer self.lConnPool.Unlock()

	for componentName, _ := range self.ConnPool {
		if self.ConnPool[componentName] == nil {
			self.ConnPool[componentName] = make(ComponentConns)
		}
		findComponentInterface := common.ComponentMap["Find"]
		findComponent, _ := findComponentInterface.(*Find)
		ips, err := findComponent.FindAllComponent(componentName)
		if err != nil {
			continue
		}
		for ip, conn := range self.ConnPool[componentName] {
			if _, ok := ips[ip]; ok {
				continue
			}
			self.deleteConnNoLock(&conn)
		}
		for ip, _ := range ips {
			if _, ok := self.ConnPool[componentName][ip]; ok {
				continue
			}

			conn, err := grpc.Dial(ip, grpc.WithBlock(), grpc.WithInsecure(), grpc.WithKeepaliveParams(kacp))
			if err != nil {
				continue
			}

			componentConn := ComponentConn{}
			componentConn.Conn = conn
			componentConn.ComponentName = componentName
			componentConn.Ip = ip
			self.ConnPool[componentName][ip] = componentConn
			break
		}
	}

	//common.LogDebug("GRPC RefreshConn ok", self.ConnPool)
}
