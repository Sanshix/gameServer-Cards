package base

import (
	"reflect"
	"runtime/debug"

	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"github.com/golang/protobuf/proto"
)

func init() {
	common.AllComponentMap["Route"] = &Route{}
}

// Route 路由组件（用于服务之间的交互），调用其他组件服务都通过route组件
type Route struct {
	common.RouteI
	Base
}

func (self *Route) LoadComponent(config *common.OneComponentConfig, componentName string) {
	self.Base.LoadComponent(config, componentName)

	return
}

func (self *Route) CallAnyReply(componentName string, methodName string, request proto.Message, extroInfo *pb.MessageExtroInfo) (proto.Message, *pb.ErrorMessage) {
	defer func() {
		if err := recover(); err != nil {
			common.LogError("Route CallAnyReply has err", err)
			debug.PrintStack()
		}
	}()
	reply, err := self.realCall(componentName, methodName, request, extroInfo)
	return reply, err
}

func (self *Route) Call(componentName string, methodName string, request proto.Message, reply proto.Message, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage {
	defer func() {
		if err := recover(); err != nil {
			common.LogError("Route Call has err", err)
			debug.PrintStack()
		}
	}()
	replyTemp, err := self.realCall(componentName, methodName, request, extroInfo)
	if err != nil {
		return err
	}
	reply.Reset()
	proto.Merge(reply, replyTemp)
	return nil
}

// 调用内部组件或者转发给GRPC与外部服务交互
func (self *Route) realCall(componentName string, methodName string, request proto.Message, extroInfo *pb.MessageExtroInfo) (proto.Message, *pb.ErrorMessage) {
	var reply proto.Message
	if common.ComponentMap[componentName] != nil {
		// 内部组件
		methodArgs := []reflect.Value{reflect.ValueOf(request), reflect.ValueOf(extroInfo)}
		rst := reflect.ValueOf(common.ComponentMap[componentName]).MethodByName(methodName).Call(methodArgs)
		if rst[0].Interface() != nil {
			reply = rst[0].Interface().(proto.Message)
		}
		var err *pb.ErrorMessage
		if rst[1].Interface() != nil {
			err = rst[1].Interface().(*pb.ErrorMessage)
		} else {
			err = nil
		}
		return reply, err
	}
	// 外部服务
	grpcComponentInterface := common.ComponentMap["GRPC"]
	grpcComponent, _ := grpcComponentInterface.(*GRPC)
	reply, err := grpcComponent.SendMessage(componentName, methodName, request, extroInfo)
	if err != nil {
		return reply, err
	}
	return reply, nil
}

// 给指定ip地址的服务发送
func (self *Route) CallByIp(ip string, componentName string, methodName string, request proto.Message, reply proto.Message, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage {
	defer func() {
		if err := recover(); err != nil {
			common.LogError("Route CallByIp has err", err)
			debug.PrintStack()
		}
	}()
	replyTemp, err := self.realCallByIp(ip, componentName, methodName, request, extroInfo)
	if err != nil {
		return err
	}
	reply.Reset()
	proto.Merge(reply, replyTemp)
	return nil
}

// 通过GRPC组件转发给其他服务
func (self *Route) realCallByIp(ip string, componentName string, methodName string, request proto.Message, extroInfo *pb.MessageExtroInfo) (proto.Message, *pb.ErrorMessage) {
	var reply proto.Message
	grpcComponentInterface := common.ComponentMap["GRPC"]
	grpcComponent, _ := grpcComponentInterface.(*GRPC)
	reply, err := grpcComponent.
		SendMessageByIp(ip, componentName, methodName, request, extroInfo)
	if err != nil {
		return reply, err
	}
	return reply, nil
}
