package common

import (
	pb "gameServer-demo/src/grpc"
	"runtime/debug"
	"sync"
	"time"

	socketio "github.com/googollee/go-socket.io"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

// ClientManager 链接管理器
type ClientManager struct {
	// 链接
	Clients map[string]*Client
	// 管理器锁
	LManager sync.Mutex
	// 服务器ip
	ServerIP string
	// grpc端口
	GrpcPort string
}

// Client 链接
type Client struct {
	// 链接id
	ID string
	// 链接用户id
	UserID string
	// 链接的socket结构体
	Socket socketio.Socket
	// 是否是已关闭的链接
	IsClosed bool
	// 发送信息的通道
	Send chan []byte
	// 链接锁
	LConn sync.Mutex
	// 链接开始的时间
	StartTime int64
	// 链接玩家的角色
	Role pb.Roles
	// 链接玩家的权限
	AuthMap map[pb.AuthorizationDef]pb.AuthorizationDef
}

// getClientByUserID 通过用户id获得链接
func (m *ClientManager) getClientByUserID(userID string) *Client {
	for _, client := range m.Clients {
		if client.UserID == userID {
			return client
		}
	}
	return nil
}

// RegisterClient 注册一个新的链接
func (m *ClientManager) RegisterClient(c *Client) {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	m.Clients[c.ID] = c
	//LogInfo("ClientManager: registerClient", c.ID)
}

// Clear 清除全部链接
func (m *ClientManager) Clear() {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	for _, client := range m.Clients {
		client.Close()
	}
	m.Clients = make(map[string]*Client)
}

// UserLoginOk 玩家登陆成功回掉
func (m *ClientManager) UserLoginOk(message *pb.EmptyMessage, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	reply := &pb.EmptyMessage{}
	userID := extroInfo.GetUserId()
	clientConnID := extroInfo.GetClientConnId()
	//LogInfo("ClientManager UserLoginOk in", clientConnID, userID, extroInfo.Role, extroInfo.Auths)
	// 检测链接id是否有效
	if _, ok := m.Clients[clientConnID]; !ok {
		LogError("ClientManager UserLoginOk invalid clientId", clientConnID, userID)
		return reply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	client := m.Clients[clientConnID]
	// 更新链接数据
	msgErr := client.Update(userID, extroInfo)
	if msgErr != nil {
		LogError("ClientManager UserLoginOk client.Update has err", clientConnID, userID)
		return reply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	LogInfo("ClientManager UserLoginOk all ok", clientConnID, userID, extroInfo.Role, extroInfo.Auths)

	return reply, nil
}

// Update 更新链接数据
func (c *Client) Update(userID string, extroInfo *pb.MessageExtroInfo) *pb.ErrorMessage {
	c.LConn.Lock()
	defer c.LConn.Unlock()

	// 如果链接已经关闭或者等待关闭，则报错
	if c.IsClosed == true {
		LogError("Client Update client already closed", userID)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	// 这个链接已经有玩家了
	if c.UserID != "" {
		LogError("Client Update client already has user", userID)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	// 绑定用户mq队列
	err := MQer.BindUser(userID, func(msg []byte) error {
		c.SendByteMessage(msg)
		return nil
	})
	if err != nil {
		LogError("Client Update MQer.BindUser has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, err.Error())
	}
	// 设置玩家在线
	msgErr := Pusher.SetOnline(userID, ServerIndex)
	if msgErr != nil {
		LogError("Client Update SetOnline has err", msgErr)
		return msgErr
	}
	// 设置链接数据
	c.UserID = userID
	c.Role = extroInfo.Role
	authMap := map[pb.AuthorizationDef]pb.AuthorizationDef{}
	authArr := extroInfo.Auths
	for _, auth := range authArr {
		authMap[auth] = auth
	}
	c.AuthMap = authMap
	return nil
}

// UpdateClient
/*func (m *ClientManager) UpdateClient(clientId string, userId string) (*Client, error) {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	if _, ok := m.Clients[userId]; ok {
		return nil, errors.New("WebSocket updateClient userId already in clients:" + userId)
	}
	if _, ok := m.Clients[clientId]; !ok {
		LogInfo("UpdateClient m.Clients[clientId]", m.Clients)
		return nil, errors.New("WebSocket updateClient clientId err," + clientId)
	}
	m.Clients[userId] = m.Clients[clientId]
	delete(m.Clients, clientId)
	return m.Clients[userId], nil
}*/

/*func (m *ClientManager) GetClientByConnId(connId string) *Client {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	if _, ok := m.Clients[connId]; ok {
		return m.Clients[connId]
	}
	return nil
}*/

/*func (m *ClientManager) ValidClient(c *Client) bool {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	if _, ok := m.Clients[c.Id]; ok {
		return true
	}
	if _, ok := m.Clients[c.UserId]; ok {
		return true
	}
	return false
}*/
// 注销客户端
func (m *ClientManager) unregisterClient(c *Client) {
	delete(m.Clients, c.ID)
	c.Close()

	//LogInfo("ClientManager unregisterClient", c.UserID, c.ID)
}

// UnregisterClient 注销客户端
func (m *ClientManager) UnregisterClient(c *Client) {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	m.unregisterClient(c)

	//LogInfo("ClientManager: UnregisterClient", c.UserID, c.ID)
}

// Push 服务器推送
func (m *ClientManager) Push(message proto.Message, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	reply := &pb.EmptyMessage{}
	userID := extroInfo.GetUserId()
	client := m.getClientByUserID(userID)
	if client == nil {
		return reply, nil
	}

	replyMessageName := proto.MessageName(message)
	clientReplyContent, err := ptypes.MarshalAny(message)
	if err != nil {
		LogError("ClientManager Push MarshalAny has err", err)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, err.Error())
	}
	go client.SendMessage(replyMessageName, clientReplyContent)
	return reply, nil
}

// Kick 服务器踢人
func (m *ClientManager) Kick(message proto.Message, extroInfo *pb.MessageExtroInfo, connectionCloseCall func(connectionClient *Client)) (*pb.EmptyMessage, *pb.ErrorMessage) {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	reply := &pb.EmptyMessage{}
	userID := extroInfo.GetUserId()
	clientConnID := extroInfo.GetClientConnId()
	var client *Client
	if clientConnID != "" {
		if _, ok := m.Clients[clientConnID]; !ok {
			LogInfo("ClientManager Kick clientConn clientConnID not online:", userID, clientConnID)
			return reply, nil
		}
		client = m.Clients[clientConnID]
	} else if userID != "" {
		client = m.getClientByUserID(userID)
		if client == nil {
			LogInfo("ClientManager Kick clientConn userID not online:", userID, clientConnID)
			return reply, nil
		}
		clientConnID = client.ID
	}
	/*if _, ok := m.Clients[clientConnId]; !ok {
		LogInfo("WebSocket Kick clientConn not online:", userId, clientConnId)
		return reply, nil
	}
	var client = m.Clients[clientConnId]
	m.UnregisterClient(client)
	go client.Close(func() {
		connectionCloseCall(client)
	})*/
	m.unregisterClient(client)
	return reply, nil
	//return reply, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "WebSocket Kick userId or clientConnId is nil")
}

// Broadcast 服务器广播
func (m *ClientManager) Broadcast(message proto.Message, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	m.LManager.Lock()
	defer m.LManager.Unlock()

	reply := &pb.EmptyMessage{}
	for _, conn := range m.Clients {
		replyMessageName := proto.MessageName(message)
		clientReplyContent, err := ptypes.MarshalAny(message)
		if err != nil {
			LogInfo("ClientManager Broadcast MarshalAny has err", err)
			continue
		}
		go conn.SendMessage(replyMessageName, clientReplyContent)
	}
	return reply, nil
}

// CloseChan 关闭chan
func (c *Client) closeChan() {
	defer func() {
		if recover() != nil {
			// close(ch) panic occur
		}
	}()

	close(c.Send) // panic if ch is closed
}

// Close 关闭链接
func (c *Client) Close() {
	c.LConn.Lock()
	defer c.LConn.Unlock()

	if c.IsClosed == true {
		return
	}
	c.IsClosed = true
	userID := c.UserID
	if userID != "" {
		extroInfo := &pb.MessageExtroInfo{}
		playerMutex, err := Locker.MessageLock(MessageLockPlayer+userID, extroInfo, "Client")
		if err != nil {
			LogError("Client Close  MessageLockPlayer has err", err)
			return
		}
		defer Locker.MessageUnlock(MessageLockPlayer+userID, extroInfo, "Client", playerMutex)
		MQer.UnBindUser(userID)
		Pusher.SetOffline(userID)
	}
	//LogInfo("Client close", c.UserID, c.ID)
	c.Socket.Disconnect()
	c.closeChan()
	c.UserID = ""
	c.Role = pb.Roles_Error
	c.AuthMap = nil
}

// SendMessage 发送消息
func (c *Client) SendMessage(messageName string, message *any.Any) {
	defer func() {
		if err := recover(); err != nil {
			LogError("WebSocket sendMessage has err", err)
			debug.PrintStack()
		}
	}()

	c.LConn.Lock()
	defer c.LConn.Unlock()

	if c.IsClosed == true {
		return
	}

	/*if c.userId == "" {
		return
	}*/

	clientReply := &pb.ClientReply{}
	clientReply.MessageName = messageName
	clientReply.MessageContent = message
	byteInfo, err := proto.Marshal(clientReply)
	if err != nil {
		LogError("Client: sendMessage has err", c.ID, c.UserID, err)
		return
	}

	c.Send <- byteInfo
}

// SendByteMessage 发送消息
func (c *Client) SendByteMessage(message []byte) {
	defer func() {
		if err := recover(); err != nil {
			LogError("Client SendByteMessage has err", err)
			debug.PrintStack()
		}
	}()

	c.LConn.Lock()
	defer c.LConn.Unlock()

	if c.IsClosed == true {
		return
	}

	c.Send <- message
}

// ReceiveMessage 收到消息
func (c *Client) ReceiveMessage(manager *ClientManager, message []byte) *pb.ErrorMessage {
	defer func() {
		if err := recover(); err != nil {
			LogError("WebSocket receiveMessage has err", err)
			debug.PrintStack()
		}
	}()

	c.LConn.Lock()
	isClientLock := true
	defer func() {
		if isClientLock == true {
			c.LConn.Unlock()
		}
	}()

	messageStartTime := time.Now().UnixNano() / 1e6

	clientRequest := &pb.ClientRequest{}
	err := proto.Unmarshal(message, clientRequest)
	if err != nil {
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, err.Error())
	}
	componentName := clientRequest.GetComponentName()
	methodName := clientRequest.GetMethodName()
	if c.Role == pb.Roles_Error {
		c.Role = pb.Roles_Guest
	}
	//c.AuthMap = make([]pb.AuthorizationDef,0)
	if !Authorizationer.Verify(componentName+"_"+methodName, c.Role, c.AuthMap) {
		errMsg := "method not found: " + componentName + "_" + methodName
		LogError("WebSocketComponent: receiveMessage method not found", c.ID, c.UserID, componentName, methodName, c.Role, c.AuthMap)
		return GetGrpcErrorMessage(pb.ErrorCode_AuthorizationDenied, errMsg)
	}

	clientMessage, err := ptypes.Empty(clientRequest.GetMessageContent())
	if err != nil {
		LogError("WebSocketComponent ptypes.Empty has err", c.ID, c.UserID, err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, err.Error())
	}
	err = ptypes.UnmarshalAny(clientRequest.GetMessageContent(), clientMessage)
	if err != nil {
		LogError("WebSocketComponent ptypes.UnmarshalAny has err", c.ID, c.UserID, err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, err.Error())
	}

	messageExtroInfo := &pb.MessageExtroInfo{}
	messageExtroInfo.ClientConnId = c.ID
	messageExtroInfo.ClientConnIp = manager.ServerIP + ":" + manager.GrpcPort
	messageExtroInfo.UserId = c.UserID
	messageExtroInfo.Role = c.Role
	messageExtroInfo.Auths = make([]pb.AuthorizationDef, 0)
	messageExtroInfo.ServerIndex = ServerIndex
	if nil != c.AuthMap {
		for _, auth := range c.AuthMap {
			messageExtroInfo.Auths = append(messageExtroInfo.Auths, auth)
		}
	}
	//LogInfo("Client before lock", c.UserID, componentName, methodName)
	// 在这里就加玩家自己的锁
	if c.UserID != "" {
		playerMutex, err := Locker.MessageLock(MessageLockPlayer+c.UserID, messageExtroInfo, "socketManager")
		if err != nil {
			LogError("WebSocketComponent MessageLock has err", err)
			return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		defer Locker.MessageUnlock(MessageLockPlayer+c.UserID, messageExtroInfo, "socketManager", playerMutex)
	} else {
		// 还没有登陆时只接受login组件的协议
		if componentName != "Login" && componentName != "WeChat" && componentName != "Facebook" {
			LogError("WebSocketComponent only accpet Login when user not login", componentName)
			return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		// 并解锁
		c.LConn.Unlock()
		isClientLock = false
	}
	//LogInfo("Client after lock", c.UserID)

	reply, msgErr := Router.CallAnyReply(componentName, methodName, clientMessage, messageExtroInfo)
	if msgErr != nil {
		LogError("WebSocketComponent CallAnyReply has err", componentName, methodName, c.ID, c.UserID, msgErr)
		return c.SendErrorMsg(msgErr)
	}
	replyMessageName := proto.MessageName(reply)
	clientReplyContent, err := ptypes.MarshalAny(reply)
	if err != nil {
		LogError("WebSocketComponent MarshalAny has err", c.ID, c.UserID, err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, err.Error())
	}
	go c.SendMessage(replyMessageName, clientReplyContent)

	messageEndTime := time.Now().UnixNano() / 1e6
	costTime := messageEndTime - messageStartTime
	LogDebug("message handle ok", costTime, componentName, methodName)

	return nil
}

func (c *Client) SendErrorMsg(errorMsg *pb.ErrorMessage) *pb.ErrorMessage {
	replyMessageName := proto.MessageName(errorMsg)
	clientReplyContent, err := ptypes.MarshalAny(errorMsg)
	if err != nil {
		LogInfo("SendErrorMsg MarshalAny has err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, err.Error())
	}
	go c.SendMessage(replyMessageName, clientReplyContent)
	return nil
}
