package base

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	socketio "github.com/googollee/go-socket.io"
	uuid "github.com/satori/go.uuid"
)

func init() {
	common.AllComponentMap["SocketIO"] = &SocketIO{}
}

// SocketIO 使用 go-socket.io
type SocketIO struct {
	Base
	manager             common.ClientManager
	server              *socketio.Server
	clearConnectionTime int
}

func index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello word!"))
}

type customServer struct {
	Server *socketio.Server
}

func (s *customServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//common.LogInfo("customServer ServeHTTP in")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	origin := r.Header.Get("Origin")
	w.Header().Set("Access-Control-Allow-Origin", origin)
	s.Server.ServeHTTP(w, r)
}

func (w *SocketIO) LoadComponent(config *common.OneComponentConfig, componentName string) {
	w.Base.LoadComponent(config, componentName)
	w.manager = common.ClientManager{
		Clients: make(map[string]*common.Client),
	}
	conn, err := net.Dial("udp", "www.google.com.hk:80")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	w.manager.ServerIP = strings.Split(conn.LocalAddr().String(), ":")[0]
	w.manager.GrpcPort = (*w.Config)["grpc_port"]
	clearConnectionTime := (*w.Config)["clear_connection_time"]
	w.clearConnectionTime, err = strconv.Atoi(clearConnectionTime)
	if err != nil {
		panic(err)
	}

	server, err := socketio.NewServer(nil)
	w.server = server

	customServerObj := new(customServer)
	customServerObj.Server = server

	w.configureSocketIO()
	//go server.Serve()

	go func() {
		//http.Handle("/", corsMiddleware(http.HandlerFunc(index)))
		//http.Handle("/socket.io/", corsMiddleware(w.server))
		http.Handle("/socket.io/", customServerObj)
		http.ListenAndServe((*w.Config)["listen_url"], nil)

	}()
	common.LogInfo("SocketIO listen on :", (*w.Config)["listen_url"])
	return
}

func (w *SocketIO) Start() {
	initGlobleConfigNameArr := []string{
		"SocketServerNum",
	}
	err := common.InitGlobleConfigTemp(initGlobleConfigNameArr)
	if err != nil {
		panic(err)
	}
}

func (w *SocketIO) Clear() {
	w.manager.Clear()
}

func (w *SocketIO) UserLoginOk(message *pb.EmptyMessage, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	return w.manager.UserLoginOk(message, extroInfo)
}

func (w *SocketIO) Push(message proto.Message, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	return w.manager.Push(message, extroInfo)
}

func (w *SocketIO) Kick(message proto.Message, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	return w.manager.Kick(message, extroInfo, func(client *common.Client) {
		client.Socket.(socketio.Socket).Disconnect()
	})
}

func (w *SocketIO) Broadcast(message proto.Message, extroInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	return w.manager.Broadcast(message, extroInfo)
}

// corsMiddleware 跨域设置中间件
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowHeaders := "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization"

		//w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, PUT, PATCH, GET, DELETE")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", allowHeaders)

		next.ServeHTTP(w, r)
	})
}

func (w *SocketIO) clientWrite(c *common.Client) {
	defer func() {
		if err := recover(); err != nil {
			common.LogError("SocketIO clientWrite has err", err)
			debug.PrintStack()
		}
		w.manager.UnregisterClient(c)
		/*c.Close(func() {
			c.Socket.(socketio.Socket).Disconnect()
		})*/
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				return
			}
			go c.Socket.(socketio.Socket).Emit("msg", common.ByteArrayToMsg(message))
		}
	}
}

// configureSocketIO 配置SocketIO 收发事件
func (w *SocketIO) configureSocketIO() {
	w.server.On("connection", func(so socketio.Socket) {
		common.LogInfo("SocketIO: connection in", so.Id())
		defer func() {
			if err := recover(); err != nil {
				common.LogError("configureSocketIO error", err)
				debug.PrintStack()
			}
		}()
		connUUID := uuid.NewV4()

		client := &common.Client{
			ID:        connUUID.String(), //connUUID.String(),
			UserID:    "",
			Send:      make(chan []byte),
			StartTime: time.Now().Unix(),
			IsClosed:  false,
		}

		so.On("msg", func(message string) {
			realMessage := common.MsgToByteArray(message)
			//common.LogInfo("OnEvent", realMessage)
			err1 := client.ReceiveMessage(&w.manager, realMessage)
			if err1 != nil {
				common.LogError("SocketIO clientRead receiveMessage has err", err1)
				client.SendErrorMsg(err1)
				return
			}
		})
		so.On("net_ping", func() {
			go so.Emit("net_pong", "")
		})
		so.On("disconnection", func() {
			common.LogInfo("SocketIO: disconnection", so.Id(), client.ID)
			w.manager.UnregisterClient(client)
			/*client.Close(func() {
				so.Disconnect()
			})*/
		})

		//10秒内没有登陆成功就关闭这个链接
		common.StartTimer(time.Second*time.Duration(w.clearConnectionTime), false, func() bool {
			if client.UserID == "" {
				common.LogInfo("SocketIO socket long time not login:", so.Id(), client.ID)
				//self.clientClose(client)
				w.manager.UnregisterClient(client)
				/*client.Close(func() {
					so.Disconnect()
				})*/
			}
			return false
		})

		client.Socket = so
		w.manager.RegisterClient(client)
		//common.LogInfo("SocketIO: Client open", so.Id(), client.ID, client.UserID)
		go w.clientWrite(client)
	})
	w.server.On("error", func(so socketio.Socket, err error) {
		common.LogInfo("SocketIO: on error", so.Id(), err)
	})
}
