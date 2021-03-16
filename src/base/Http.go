package base

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"time"

	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
)

func init() {
	common.LogDebug("❤Http服务初始化❤")
	common.AllComponentMap["Http"] = &Http{}
}

type Http struct {
	Base
}

// TODO http 包改成gin包
func (self *Http) LoadComponent(config *common.OneComponentConfig, componentName string) {
	self.Base.LoadComponent(config, componentName)
	go func() {
		http.HandleFunc("/proto/api", self.httpHandler)
		http.HandleFunc("/health", self.healthHandler)
		// 监听http端口
		http.ListenAndServe((*self.Config)["listen_url"], nil)
	}()

	return
}

func (self *Http) healthHandler(w http.ResponseWriter, r *http.Request) {
	common.LogInfo("Http healthHandler data in")
	defer func() {
		if err := recover(); err != nil {
			common.LogError("Http healthHandler has err", err)
			http.Error(w, "", http.StatusInternalServerError)
			debug.PrintStack()
		}
	}()

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST")
	w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(200)
	fmt.Fprintln(w, "OK")
}

func (self *Http) httpHandler(w http.ResponseWriter, r *http.Request) {
	//common.LogInfo("Http httpHandler data in")
	defer func() {
		if err := recover(); err != nil {
			common.LogError("Http httpHandler has err", err)
			http.Error(w, "", http.StatusInternalServerError)
			debug.PrintStack()
		}
	}()
	messageStartTime := time.Now().UnixNano() / 1e6

	w.Header().Set("Access-Control-Allow-Origin", "*")
	//w.Header().Add("Access-Control-Allow-Headers", "Content-Type")
	//w.Header().Set("Content-type", "text/plain; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, GET, POST")
	ioR, err := ioutil.ReadAll(r.Body)
	//common.LogDebug("Http httpHandler ioR 1", ioR, r.Header, r.Body)
	if err != nil {
		common.LogError("Http httpHandler ReadAll has err", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	//common.LogDebug("Http httpHandler ioR", ioR)
	clientRequest := &pb.ClientRequest{}
	err = proto.Unmarshal(ioR, clientRequest)
	if err != nil {
		common.LogError("Http httpHandler proto.Unmarshal has err", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	common.LogDebug("Http httpHandler clientRequest", clientRequest)

	componentName := clientRequest.GetComponentName()
	methodName := clientRequest.GetMethodName()
	if common.ClientInterfaceMap[componentName+"."+methodName] != true {
		common.LogError("Http: httpHandler method not found", componentName, methodName)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	clientMessage, err := ptypes.Empty(clientRequest.GetMessageContent())
	if err != nil {
		common.LogError("Http httpHandler ptypes.Empty has err", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	err = ptypes.UnmarshalAny(clientRequest.GetMessageContent(), clientMessage)
	if err != nil {
		common.LogError("Http httpHandler ptypes.UnmarshalAny has err", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	messageExtroInfo := &pb.MessageExtroInfo{}
	//messageExtroInfo.ClientConnId = c.id
	//messageExtroInfo.ClientConnIp = serverIp + ":" + grpcPort
	//messageExtroInfo.UserId = c.userId
	reply, msgErr := common.Router.CallAnyReply(componentName, methodName, clientMessage, messageExtroInfo)
	var replyMessageName string
	var clientReplyContent *any.Any
	if msgErr != nil {
		replyMessageName = "ErrorMessage"
		clientReplyContent, err = ptypes.MarshalAny(msgErr)
		if err != nil {
			common.LogError("Http httpHandler msgErr MarshalAny has err", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	} else {
		replyMessageName = proto.MessageName(reply)
		clientReplyContent, err = ptypes.MarshalAny(reply)
		if err != nil {
			common.LogError("Http httpHandler MarshalAny has err", err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	}

	clientReply := &pb.ClientReply{}
	clientReply.MessageName = replyMessageName
	clientReply.MessageContent = clientReplyContent
	byteInfo, err := proto.Marshal(clientReply)
	if err != nil {
		common.LogError("Http httpHandler Marshal has err", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.Write(byteInfo)

	messageEndTime := time.Now().UnixNano() / 1e6
	costTime := messageEndTime - messageStartTime
	common.LogDebug("Http httpHandler message handle ok", costTime, componentName, methodName)
}
