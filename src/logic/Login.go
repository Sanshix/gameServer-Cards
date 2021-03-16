package logic

import (
	"gameServer-demo/src/base"
	pb "gameServer-demo/src/grpc"

	"gameServer-demo/src/common"
)

func init() {
	common.AllComponentMap["Login"] = &Login{}
}

// Login 登陆组件
type Login struct {
	base.Base
}

// LoadComponent 加载组件
func (obj *Login) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)

	return
}

// Login 用户登陆
func (obj *Login) Login(request *pb.LoginRequest, extroInfo *pb.MessageExtroInfo) (*pb.LoginReply, *pb.ErrorMessage) {
	account := request.GetAccount()
	password := request.GetPassword()

	realReply, msgErr := common.Login(account, password, request.GetMobile(), pb.Roles_Player, obj.ComponentName, extroInfo, func(reply *pb.LoginReply) *pb.ErrorMessage {
		//必须用此接受空的playerInfo 否则会将playerInfo信息清空
		if common.SelectComponentExist("Gift") {
			PlayerInfoNone := &pb.PlayerInfo{}
			msgErr := common.Router.Call("Gift", "AddGiftForPlayer", reply.PlayerInfo, PlayerInfoNone, extroInfo)

			reply.PlayerInfo = PlayerInfoNone
			return msgErr
		}
		return nil
	})
	return realReply, msgErr
}

// ManagerLogin 管理员登录
func (obj *Login) ManagerLogin(request *pb.LoginRequest, extroInfo *pb.MessageExtroInfo) (*pb.LoginReply, *pb.ErrorMessage) {
	account := request.GetAccount()
	password := request.GetPassword()
	realReply, msgErr := common.Login(account, password, "", pb.Roles_Manager, obj.ComponentName, extroInfo, func(reply *pb.LoginReply) *pb.ErrorMessage {
		return nil
	})
	return realReply, msgErr
}
