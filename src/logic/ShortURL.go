package logic

import (
	"gameServer-demo/src/base"
	pb "gameServer-demo/src/grpc"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"gameServer-demo/src/common"
)

func init() {
	common.AllComponentMap["ShortURL"] = &ShortURL{}
}

// ShortURL 短链接组件
type ShortURL struct {
	base.Base
	registerURL  string
	openShortURL bool
	// 生成短链接的地址
	makeURL      string
	makeUserName string
	makeKey      string
}

// LoadComponent 加载组件
func (s *ShortURL) LoadComponent(config *common.OneComponentConfig, componentName string) {
	s.Base.LoadComponent(config, componentName)
	s.registerURL = (*s.Config)["register_url"]
	s.makeURL = (*s.Config)["make_url"]
	s.makeUserName = (*s.Config)["make_username"]
	s.makeKey = (*s.Config)["make_key"]
	openShortURL := (*s.Config)["open_short_url"]
	if openShortURL == "true" {
		s.openShortURL = true
	} else {
		s.openShortURL = false
	}

	common.StartTimer(5*time.Second, false, func() bool {
		s.GetShortURL(&pb.GetShortURLRequest{}, &pb.MessageExtroInfo{UserId: "11"})
		return false
	})

	return
}

// GetShortURL 获得短链接
func (s *ShortURL) GetShortURL(request *pb.GetShortURLRequest, extroInfo *pb.MessageExtroInfo) (*pb.GetShortURLReply, *pb.ErrorMessage) {
	lineCodeUUID := request.GetLineCodeUUID()
	realURL := s.registerURL
	reply := &pb.GetShortURLReply{}
	reply.LineCodeUUID = lineCodeUUID
	uuid := extroInfo.GetUserId()
	if uuid == "" {
		common.LogError("ShortURL GetShortURL uuid == nil")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotLogin, "")
	}
	// 直属链接
	if lineCodeUUID == "" {
		realURL = realURL + "?parent=" + uuid
	} else {
		// 排线链接
		realURL = realURL + "?parent=" + uuid + "&promotionLineId=" + lineCodeUUID
	}
	if s.openShortURL == false {
		reply.Url = realURL
	} else {
		data := make(url.Values)
		data["username"] = []string{s.makeUserName}
		data["key"] = []string{s.makeKey}
		mdata := "{data:[{\"link\":\"" + realURL + "\",\"title\":\"url\"}]}"
		data["mdata"] = []string{mdata}
		data["type"] = []string{"0"}
		common.LogInfo("1111", data)
		res, err := http.PostForm(s.makeURL, data)
		if err != nil {
			common.LogError("ShortURL PostForm has err", err)
			return reply, nil
		}
		defer res.Body.Close()
		common.LogInfo("2222")
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			common.LogError("ShortURL ioutil.ReadAll has err", err)
			return reply, nil
		}
		common.LogInfo("33333")
		common.LogInfo(string(body))
	}
	return reply, nil
}
