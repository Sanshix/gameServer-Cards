package base

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

func init() {
	common.AllComponentMap["Push"] = &Push{}
}

// Push 推送组件，完成消息推送以及一些玩家操作
type Push struct {
	Base
	common.PushI
}

// LoadComponent 加载组件
func (obj *Push) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)

	return
}

// UserLoginOk 用户登陆成功回掉，让socketio设置用户id
func (obj *Push) UserLoginOk(userID string, role pb.Roles, auths []pb.AuthorizationDef, clientConnID string, serverIndex string) *pb.ErrorMessage {
	request := &pb.EmptyMessage{}
	reply := &pb.EmptyMessage{}
	extroInfo := &pb.MessageExtroInfo{}
	extroInfo.UserId = userID
	extroInfo.ClientConnId = clientConnID
	extroInfo.Role = role
	extroInfo.Auths = auths
	msgErr := common.Router.Call("SocketIO"+serverIndex, "UserLoginOk", request, reply, extroInfo)
	if msgErr != nil {
		//common.LogError("Push OnLine CallByIp WebSocket UserLoginOk has err", msgErr)
		//失败了就设置为离线
		obj.SetOffline(userID)
		return msgErr
	}
	return nil
}

// SetOnline 将玩家设置为在线状态
// 此方法不加锁，请在外部保证有锁
func (obj *Push) SetOnline(uuid string, serverIndex string) *pb.ErrorMessage {
	extroInfo := &pb.MessageExtroInfo{}
	playerOnlineInfo := &pb.PlayerOnlineInfo{}
	playerOnlineInfo.LoginTime = time.Now().Unix()
	playerOnlineInfo.Uuid = uuid
	playerOnlineInfo.SocketServerIndex = serverIndex

	playerOnlineInfoByte, err := proto.Marshal(playerOnlineInfo)
	if err != nil {
		common.LogError("Push SetOnline Marshal has err", playerOnlineInfo, err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = common.RedisOnlineUserTable
	redisRequest.Key = uuid
	redisRequest.ValueByte = playerOnlineInfoByte
	redisReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "HSetByte", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		return msgErr
	}

	return nil
}

// SetOffline 将玩家设置为离线状态
// 此方法不加锁，请在外部保证有锁
func (obj *Push) SetOffline(uuid string) *pb.ErrorMessage {
	extroInfo := &pb.MessageExtroInfo{}

	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = common.RedisOnlineUserTable
	redisRequest.Key = uuid
	redisRequest.ValueStringArr = []string{uuid}
	redisReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "HDel", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		return msgErr
	}

	return nil
}

// CheckOnline 检测玩家的在线状态
// 参数：玩家的uuid
// 返回值：是否在线，rpc错误
// 此方法不加锁，请在外部保证有锁
func (obj *Push) CheckOnline(uuid string) (bool, *pb.ErrorMessage) {
	extroInfo := &pb.MessageExtroInfo{}

	redisRequest := &pb.RedisMessage{}
	redisRequest.Table = common.RedisOnlineUserTable
	redisRequest.Key = uuid
	redisReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "HGetByte", redisRequest, redisReply, extroInfo)
	if msgErr != nil {
		return false, msgErr
	}
	if redisReply.GetValueByte() == nil {
		return false, nil
	}

	return true, nil
}

// PushMany 向一组玩家推送消息
func (obj *Push) PushMany(pushMessage proto.Message, uidList []string) {
	for _, uid := range uidList {
		obj.Push(pushMessage, uid)
	}
}

// Push 向一个玩家推送消息
func (obj *Push) Push(pushMessage proto.Message, uid string) {
	replyMessageName := proto.MessageName(pushMessage)
	clientReplyContent, err := ptypes.MarshalAny(pushMessage)
	if err != nil {
		common.LogError("Push Push MarshalAny has err", err)
		return
	}
	clientReply := &pb.ClientReply{}
	clientReply.MessageName = replyMessageName
	clientReply.MessageContent = clientReplyContent
	byteInfo, err := proto.Marshal(clientReply)
	if err != nil {
		common.LogError("Push: Push Marshal has err", err)
		return
	}
	common.MQer.SendToUser(uid, byteInfo)
}

// Broadcast 全服广播
func (obj *Push) Broadcast(request proto.Message) {
	serverNumConfig := common.Configer.GetGlobal("SocketServerNum")
	serverNumStr := serverNumConfig.GetValue()
	serverNum, err := strconv.Atoi(serverNumStr)
	if err != nil {
		common.LogError("Push Broadcast Atoi(serverNumStr) has err", serverNumStr)
		return
	}
	//假设有100个服务器
	for index := 1; index <= serverNum; index++ {
		findComponentInterface := common.ComponentMap["Find"]
		findComponent, _ := findComponentInterface.(*Find)
		_, err := findComponent.FindAllComponent("SocketIO" + strconv.Itoa(index))
		if err != nil {
			return
		}
		go func(serverIndex string) {
			reply := &pb.EmptyMessage{}
			extroInfo := &pb.MessageExtroInfo{}
			realComponentName := "SocketIO" + serverIndex
			common.Router.Call(realComponentName, "Broadcast", request, reply, extroInfo)
		}(strconv.Itoa(index))
	}
}

// Kick 踢人
func (obj *Push) Kick(uid string) *pb.ErrorMessage {
	request := &pb.RedisMessage{}
	request.Table = common.RedisOnlineUserTable
	request.Key = uid
	reply := &pb.RedisMessage{}
	extroInfo := &pb.MessageExtroInfo{}
	msgErr := common.Router.Call("Redis", "HGetByte", request, reply, extroInfo)
	if msgErr != nil {
		common.LogError("Push Push Call Redis HGetByte has err", msgErr)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	if reply.GetValueByte() == nil {
		common.LogInfo("Push Kick user not online", uid)
		return nil
	}
	playerOnlineInfo := &pb.PlayerOnlineInfo{}
	err := proto.Unmarshal(reply.GetValueByte(), playerOnlineInfo)
	if err != nil {
		common.LogError("Push Kick Unmarshal has err", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	extroInfo.UserId = uid
	reply1 := &pb.EmptyMessage{}
	realComponentName := "SocketIO" + playerOnlineInfo.GetSocketServerIndex()
	msgErr = common.Router.Call(realComponentName, "Kick", request, reply1, extroInfo)
	if msgErr != nil {
		common.LogError("Push Push Call SocketIO Kick has err", msgErr)
		return msgErr
	}
	return nil
}

// KickByConnID 通过连接id踢人，通常用在未登陆成功时
func (obj *Push) KickByConnID(connID string, IP string) {
	extroInfo := &pb.MessageExtroInfo{}
	extroInfo.ClientConnId = connID
	reply1 := &pb.EmptyMessage{}
	err := common.Router.CallByIp(IP, "SocketIO", "Kick", reply1, reply1, extroInfo)
	if err != nil {
		common.LogError("Push KickByConnId CallByIp SocketIO Push Kick err", err)
		return
	}
}
