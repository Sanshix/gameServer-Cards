package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"io/ioutil"
	"net/http"
	"strings"
)

func init() {
	common.AllComponentMap["GetPlayerInfo"] = &GetPlayerInfo{}
}

// GetPlayerInfo 玩家信息  组件
type GetPlayerInfo struct {
	base.Base
}

// LoadComponent 加载组件
func (g *GetPlayerInfo) LoadComponent(config *common.OneComponentConfig, componentName string) {
	g.Base.LoadComponent(config, componentName)
	return
}

// Start 开启组件
func (g *GetPlayerInfo) Start() {
}

// 根据shortId 获取玩家信息
func (g *GetPlayerInfo) GetPlayerInfoByShortId(request *pb.GetPlayerInfoByShortIdRequest, extra *pb.MessageExtroInfo) (*pb.GetPlayerInfoByShortIdReply, *pb.ErrorMessage) {
	reply := &pb.GetPlayerInfoByShortIdReply{}
	reply.PlayerInfos = make([]*pb.PlayerInfo, 0)

	for _, oneShortId := range request.ShortIds {
		// 获取每个玩家的信息
		oneAccount, MsgErr := common.GetAccountInfoByShortID(oneShortId)
		if MsgErr != nil {
			common.LogError("GetPlayerInfo GetPlayerInfoByShortId GetAccountInfoByShortID has err:", MsgErr)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_AccountNotFound, "")
		}
		loadPlayerRequest := &pb.LoadPlayerRequest{}
		loadPlayerRequest.Uuid = oneAccount.Uuid
		loadPlayerReply := &pb.LoadPlayerReply{}
		MsgErr = common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extra)
		if MsgErr != nil {
			common.LogError("GetPlayerInfo GetPlayerInfoByShortId Get PlayerInfo has err ", MsgErr)
			return reply, MsgErr
		}
		onePlayerInfo := &pb.PlayerInfo{}
		// 赋值返回值
		if loadPlayerReply.PlayerInfo.GetName() == "" {
			onePlayerInfo.Name = loadPlayerReply.PlayerInfo.GetShortId() //名字
		} else {
			onePlayerInfo.Name = loadPlayerReply.PlayerInfo.GetName()
		}
		onePlayerInfo.ShortId = loadPlayerReply.PlayerInfo.GetShortId()       //短位id
		onePlayerInfo.Uuid = loadPlayerReply.PlayerInfo.GetUuid()             //uuid
		onePlayerInfo.VipLevel = loadPlayerReply.PlayerInfo.GetVipLevel()     //vip等级
		onePlayerInfo.Sex = loadPlayerReply.PlayerInfo.GetSex()               //玩家的性别
		onePlayerInfo.HeadImgUrl = loadPlayerReply.PlayerInfo.GetHeadImgUrl() //头像地址
		onePlayerInfo.Country = loadPlayerReply.PlayerInfo.GetCountry()       //国家
		onePlayerInfo.Province = loadPlayerReply.PlayerInfo.GetProvince()     //省
		onePlayerInfo.City = loadPlayerReply.PlayerInfo.GetCity()             //市/区
		onePlayerInfo.SafeMoney = loadPlayerReply.PlayerInfo.GetSafeMoney()   //保险箱金额
		/*此处可以扩展*/
		reply.PlayerInfos = append(reply.PlayerInfos, onePlayerInfo)
	}

	return reply, nil
}

// 修改玩家头像
func (g *GetPlayerInfo) ReHeadImgUrl(request *pb.ReHeadImgUrlRequest, extra *pb.MessageExtroInfo) (*pb.ReHeadImgUrlReply, *pb.ErrorMessage) {
	reply := &pb.ReHeadImgUrlReply{}

	// uuid检测
	uuid := extra.GetUserId()
	if uuid == "" {
		common.LogError("GetPlayerInfo ReHeadImgUrl has err: uuid == nil")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotLogin, "")
	}

	// 当length < 4时，要判断是不是数字索引
	if len(request.NewHeadImgUrl) > 6 {
		// 当他是http时  尝试下
		if request.NewHeadImgUrl[:4] == "http" {
			res, err := http.Get(request.NewHeadImgUrl)
			body, err := ioutil.ReadAll(res.Body)
			if err != nil {
				common.LogError("GetPlayerInfo ReHeadImgUrl has err: invalid headImg url1")
				return reply, common.GetGrpcErrorMessage(pb.ErrorCode_InvalidImageUrl, "")
			}
			contentType := http.DetectContentType(body)
			if contentType != "image/jpeg" && contentType != "image/png" {
				common.LogError("GetPlayerInfo ReHeadImgUrl has err: invalid headImg url2")
				return reply, common.GetGrpcErrorMessage(pb.ErrorCode_NotIsImageUrl, "")
			}
		} else if request.NewHeadImgUrl[:6] == "index_" {

		} else {
			common.LogError("GetPlayerInfo ReHeadImgUrl has err: invalid headImg url3")
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_NotIsImageUrl, "")
		}
		// 其余情况判断链接是否是张图片
	} else {
		common.LogError("GetPlayerInfo ReHeadImgUrl has err: invalid headImg url4")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_NotIsImageUrl, "")
	}

	// 获取用户信息
	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extra)
	if msgErr != nil {
		common.LogError("GetPlayerInfo ReHeadImgUrl Get PlayerInfo has err:", msgErr)
		return reply, msgErr
	}

	// 将名字赋值
	loadPlayerReply.PlayerInfo.HeadImgUrl = request.GetNewHeadImgUrl()

	// 保存用户信息
	savePlayerRequest := &pb.SavePlayerRequest{
		PlayerInfo: loadPlayerReply.PlayerInfo,
		ForceSave:  false,
	}
	savePlayerReply := &pb.EmptyMessage{}
	msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, savePlayerReply, extra)
	if msgErr != nil {
		common.LogError("GetPlayerInfo ReHeadImgUrl Save PlayerInfo has err:", msgErr)
		return reply, msgErr
	}

	reply.NewHeadImgUrl = request.NewHeadImgUrl
	reply.IsSuccess = true
	return reply, nil
}

// 修改玩家名字
func (g *GetPlayerInfo) ReName(request *pb.ReNameRequest, extra *pb.MessageExtroInfo) (*pb.ReNameReply, *pb.ErrorMessage) {
	reply := &pb.ReNameReply{}

	// 检测名字长度>16不
	if len(request.NewName) > 16 {
		common.LogError("GetPlayerInfo ReName length > 16!")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_LengthNotMatched, "")
	}

	// 检测下名字是否有非法字符
	if strings.ContainsAny(request.NewName, " \\《》&*<>:：'！~!\"") { //不能有这些字符
		common.LogError("GetPlayerInfo ReName has Special Characters!")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_HasSpecialSymbol, "")
	}

	// 检测uuid
	uuid := extra.GetUserId()
	if uuid == "" {
		common.LogError("GetPlayerInfo ReName has err: uuid == nil")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_UserNotLogin, "")
	}

	// 获取用户信息
	loadPlayerRequest := &pb.LoadPlayerRequest{}
	loadPlayerRequest.Uuid = uuid
	loadPlayerReply := &pb.LoadPlayerReply{}
	msgErr := common.Router.Call("PlayerInfo", "LoadPlayer", loadPlayerRequest, loadPlayerReply, extra)
	if msgErr != nil {
		common.LogError("GetPlayerInfo ReName Get PlayerInfo has err:", msgErr)
		return reply, msgErr
	}
	// 将名字赋值
	loadPlayerReply.PlayerInfo.Name = request.GetNewName()

	// 保存用户信息
	savePlayerRequest := &pb.SavePlayerRequest{
		PlayerInfo: loadPlayerReply.PlayerInfo,
		ForceSave:  false,
	}
	savePlayerReply := &pb.EmptyMessage{}
	msgErr = common.Router.Call("PlayerInfo", "SavePlayer", savePlayerRequest, savePlayerReply, extra)
	if msgErr != nil {
		common.LogError("GetPlayerInfo ReName Save PlayerInfo has err:", msgErr)
		return reply, msgErr
	}
	reply.NewName = request.NewName
	reply.IsSuccess = true
	return reply, nil
}
