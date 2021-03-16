package base

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"

	"github.com/golang/protobuf/proto"
)

func init() {
	common.AllComponentMap["ReportPublish"] = &ReportPublish{}
}

// ReportPublish 队列发布组件（用于发送数据给MQ）
type ReportPublish struct {
	Base
}

// LoadComponent 加载组件
func (obj *ReportPublish) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *ReportPublish) Start() {
	obj.Base.Start()
}

// SendReport 发送推送消息给MQ（转成proto格式）
func (obj *ReportPublish) SendReport(reportInfo *pb.ReportMessage, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	byteInfo, err := proto.Marshal(reportInfo)
	if err != nil {
		common.LogError("ReportPublish: SendReport Marshal has err", err)
		return &pb.EmptyMessage{}, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	err = common.MQer.SendReport(byteInfo)
	if err != nil {
		common.LogError("ReportPublish: send report to MQ err", err)
		return &pb.EmptyMessage{}, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return &pb.EmptyMessage{}, nil
}
