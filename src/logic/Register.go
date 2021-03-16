package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"

	"github.com/mojocn/base64Captcha"
)

func init() {
	common.AllComponentMap["Register"] = &Register{}
}

// Register 注册组件
type Register struct {
	base.Base
	RegisterCaptcha           bool
	RegisterByLineCodeCaptcha bool
	RegisterByDirectlyCaptcha bool
	RegisterTopAgentCaptcha   bool
	RegisterSmsCaptcha        bool
}

// 验证码存储  默认:10240 个 持续10分钟
var store = base64Captcha.DefaultMemStore

// LoadComponent 加载组件
func (r *Register) LoadComponent(config *common.OneComponentConfig, componentName string) {
	r.Base.LoadComponent(config, componentName)
	// 是否开启验证码
	if (*r.Base.Config)["RegisterCaptcha"] == "true" {
		r.RegisterCaptcha = true
	}
	if (*r.Base.Config)["RegisterByLineCodeCaptcha"] == "true" {
		r.RegisterByLineCodeCaptcha = true
	}
	if (*r.Base.Config)["RegisterByDirectlyCaptcha"] == "true" {
		r.RegisterByDirectlyCaptcha = true
	}
	if (*r.Base.Config)["RegisterTopAgentCaptcha"] == "true" {
		r.RegisterTopAgentCaptcha = true
	}
	if (*r.Base.Config)["RegisterSmsCaptcha"] == "true" {
		r.RegisterSmsCaptcha = true
	}
	return
}

// Register 玩家注册
func (r *Register) Register(request *pb.RegisterRequest, extroInfo *pb.MessageExtroInfo) (*pb.RegisterReply, *pb.ErrorMessage) {
	account := request.GetAccount()
	password := request.GetPassword()
	mobile := request.GetMobile()
	//common.LogError("验证码：", request.CaptchaId, request.VerifyValue)

	if r.RegisterCaptcha { // 当设置了启动方验证
		if request.CaptchaId == "" || request.VerifyValue == "" {
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
		// 验证图片码
		if !r.verify(request.CaptchaId, request.VerifyValue) {
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
	}
	// 如果需要验证短信验证码，则验证短信
	if r.RegisterSmsCaptcha {
		if request.GetSmsCaptcha() == "" {
			common.LogError("Register Register SmsCaptcha is nil", request)
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
		if common.VerifyCaptcha(mobile, request.GetSmsCaptcha()) == false {
			common.LogError("Register Register VerifyCaptcha is false", request)
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
		// 短信注册，账号和手机号是一致的
		account = mobile
	} else {
		// 没开短信验证的话，手机号字段置空,避免bug
		mobile = ""
	}

	return common.Register(account, password, mobile, pb.Roles_Player, r.ComponentName, extroInfo, nil)
}

// RegisterRobot 机器人注册
func (r *Register) RegisterRobot(request *pb.RegisterRequest, extroInfo *pb.MessageExtroInfo) (*pb.RegisterReply, *pb.ErrorMessage) {
	account := request.GetAccount()
	password := request.GetPassword()
	return common.Register(account, password, "", pb.Roles_Robot, r.ComponentName, extroInfo, nil)
}

// RegisterByLineCode 通过排线码的代理注册
func (r *Register) RegisterByLineCode(request *pb.RegisterRequest, extroInfo *pb.MessageExtroInfo) (*pb.RegisterReply, *pb.ErrorMessage) {
	account := request.GetAccount()
	password := request.GetPassword()
	lineCodeUUID := request.GetLineCodeUUID()
	parentUUID := request.GetParentUUID()
	reply := &pb.RegisterReply{}
	if account == "" || password == "" || lineCodeUUID == "" || parentUUID == "" {
		common.LogError("Register RegisterByLineCode InvalidParameters", request)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_InvalidParameters, "")
	}
	if r.RegisterByLineCodeCaptcha { // 当设置了启动方验证
		if request.CaptchaId == "" || request.VerifyValue == "" {
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
		// 验证图片码
		if !r.verify(request.CaptchaId, request.VerifyValue) {
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
	}
	return common.Register(account, password, "", pb.Roles_Player, r.ComponentName, extroInfo, func(mysqlAccountInfo *pb.MysqlAccountInfo) *pb.ErrorMessage {
		createAgentRelationRequest := &pb.CreateAgentRelationRequest{}
		createAgentRelationRequest.LineCodeUUID = lineCodeUUID
		createAgentRelationRequest.ParentUUID = parentUUID
		createAgentRelationRequest.ChildUUID = mysqlAccountInfo.GetUuid()
		createAgentRelationReply := &pb.CreateAgentRelationReply{}
		msgErr := common.Router.Call("AgentRelationWrite", "CreateByLineCode", createAgentRelationRequest, createAgentRelationReply, extroInfo)
		if msgErr != nil {
			common.LogError("Register RegisterByLineCode Call AgentRelationWrite CreateByLineCode has err", mysqlAccountInfo, msgErr)
			return msgErr
		}
		return nil
	})
}

// RegisterByDirectly 通过直属码的代理注册
func (r *Register) RegisterByDirectly(request *pb.RegisterRequest, extroInfo *pb.MessageExtroInfo) (*pb.RegisterReply, *pb.ErrorMessage) {
	account := request.GetAccount()
	password := request.GetPassword()
	parentUUID := request.GetParentUUID()
	reply := &pb.RegisterReply{}
	if account == "" || password == "" || parentUUID == "" {
		common.LogError("Register RegisterByDirectly InvalidParameters", request)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_InvalidParameters, "")
	}
	if r.RegisterByDirectlyCaptcha { // 当设置了启动方验证
		if request.CaptchaId == "" || request.VerifyValue == "" {
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
		// 验证图片码
		if !r.verify(request.CaptchaId, request.VerifyValue) {
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
	}
	return common.Register(account, password, "", pb.Roles_Player, r.ComponentName, extroInfo, func(mysqlAccountInfo *pb.MysqlAccountInfo) *pb.ErrorMessage {
		createAgentRelationRequest := &pb.CreateAgentRelationRequest{}
		createAgentRelationRequest.ParentUUID = parentUUID
		createAgentRelationRequest.ChildUUID = mysqlAccountInfo.GetUuid()
		createAgentRelationReply := &pb.CreateAgentRelationReply{}
		msgErr := common.Router.Call("AgentRelationWrite", "Create", createAgentRelationRequest, createAgentRelationReply, extroInfo)
		if msgErr != nil {
			common.LogError("Register RegisterByDirectly Call AgentRelationWrite Create has err", mysqlAccountInfo, msgErr)
			return msgErr
		}
		return nil
	})
}

// RegisterTopAgent 注册顶代
func (r *Register) RegisterTopAgent(request *pb.RegisterRequest, extroInfo *pb.MessageExtroInfo) (*pb.RegisterReply, *pb.ErrorMessage) {
	account := request.GetAccount()
	password := request.GetPassword()
	reply := &pb.RegisterReply{}
	if account == "" || password == "" {
		common.LogError("Register RegisterTopAgent InvalidParameters", request)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_InvalidParameters, "")
	}
	if r.RegisterTopAgentCaptcha { // 当设置了启动方验证
		if request.CaptchaId == "" || request.VerifyValue == "" {
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
		// 验证图片码
		if !r.verify(request.CaptchaId, request.VerifyValue) {
			return nil, common.GetGrpcErrorMessage(pb.ErrorCode_CaptchaCodeErr, "")
		}
	}
	return common.Register(account, password, "", pb.Roles_Player, r.ComponentName, extroInfo, func(mysqlAccountInfo *pb.MysqlAccountInfo) *pb.ErrorMessage {
		setTopAgentRequest := &pb.SetTopAgentRequest{}
		setTopAgentRequest.TopUUID = mysqlAccountInfo.GetUuid()
		setTopAgentReply := &pb.SetTopAgentReply{}
		msgErr := common.Router.Call("AgentRelationWrite", "SetTopAgent", setTopAgentRequest, setTopAgentReply, extroInfo)
		if msgErr != nil {
			common.LogError("Register RegisterTopAgent Call AgentRelationWrite SetTopAgent has err", mysqlAccountInfo, msgErr)
			return msgErr
		}
		/*newPlayerRequest := &pb.NewPlayerRequest{}
		newPlayerRequest.Uuid = mysqlAccountInfo.GetUuid()
		newPlayerRequest.ShortId = mysqlAccountInfo.GetShortId()
		newPlayerRequest.RoleType = mysqlAccountInfo.GetRoleType()
		newPlayerRequest.Account = mysqlAccountInfo.GetAccount()
		newPlayerReply := &pb.NewPlayerReply{}
		msgErr = common.Router.Call("PlayerInfo", "NewPlayer", newPlayerRequest, newPlayerReply, extroInfo)
		if msgErr != nil {
			common.LogError("Register RegisterTopAgent Call PlayerInfo NewPlayer has err", msgErr)
			return msgErr
		}*/
		return nil
	})
}

// 获取验证码图片
func (r *Register) GetCaptchaPhoto(request *pb.GetCaptchaRequest, extra *pb.MessageExtroInfo) (*pb.GetCaptchaReply, *pb.ErrorMessage) {
	reply := &pb.GetCaptchaReply{}

	// 生成默认数字
	driver := base64Captcha.DefaultDriverDigit
	// 生成base64图片
	c := base64Captcha.NewCaptcha(driver, store)

	// 获取
	id, b64s, err := c.Generate()
	if err != nil {
		common.LogError("Register GetCaptchaPhoto get base64Captcha has err:", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.CaptchaId = id
	reply.Data = b64s
	return reply, nil
}

// 验证 验证码
func (r *Register) verify(id string, val string) bool {
	// 同时在内存清理掉这个图片
	return store.Verify(id, val, true)
}
