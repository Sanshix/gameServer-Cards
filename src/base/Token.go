package base

import (
	"encoding/json"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"
)

func init() {
	common.AllComponentMap["Token"] = &Token{}
}

type tokenMapData struct {
	UserUUID   string
	CreateTime int64
	TokenInfo  *common.TokenInfo
}

type Token struct {
	common.TokenI
	Base
	ExpiresIn int64
}

// LoadComponent 加载组件
func (t *Token) LoadComponent(config *common.OneComponentConfig, componentName string) {
	t.Base.LoadComponent(config, componentName)
	expiresInStr := (*t.Config)["expires_in"]
	if expiresInStr == "" {
		t.ExpiresIn = 10 * 60
		return
	}

	i64, err := strconv.ParseInt(expiresInStr, 10, 64)
	if err != nil {
		common.LogError("convert expires_in str error", err)
		i64 = 10 * 60
	}
	t.ExpiresIn = i64
	common.LogInfo("Token LoadComponent ok")
	return
}

func (t *Token) SaveThirdPartyToken(userUUID string, tokenInfo *common.TokenInfo) *pb.ErrorMessage {
	saveInfo := &tokenMapData{}

	tokenInfo.ExpiresIn = t.ExpiresIn
	saveInfo.CreateTime = time.Now().Unix()
	saveInfo.TokenInfo = tokenInfo
	saveInfo.UserUUID = userUUID
	mashData, err := json.Marshal(saveInfo)
	if err != nil {
		common.LogError("mashal token data err", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	return hSetData(common.RedisTokenMap, tokenInfo.AccessToken, mashData, &pb.MessageExtroInfo{})

}

//验证token
func (t *Token) ValidateToken(accessToken string) (*common.TokenInfo, *pb.ErrorMessage) {
	now := time.Now().Unix()
	//获取token
	tokenBytes, err := hGetData(common.RedisTokenMap, accessToken)
	if err != nil {
		return nil, err
	}
	tokenInfo := &tokenMapData{}
	err1 := json.Unmarshal(tokenBytes, tokenInfo)
	if err1 != nil {
		common.LogError("unmashal token data err", err1)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	if nil == tokenInfo.TokenInfo || tokenInfo.CreateTime+tokenInfo.TokenInfo.ExpiresIn < now {
		//移除token
		err := hDel(common.RedisTokenMap, accessToken)
		if err != nil {
			common.LogError("delete token error", err)
		}
		return nil, nil
	}
	return tokenInfo.TokenInfo, nil
}

// hSetData 将数据保存到redis的hash中
func hSetData(table string, key string, message []byte, extraInfo *pb.MessageExtroInfo) *pb.ErrorMessage {
	redisReq := &pb.RedisMessage{}
	redisReq.Table = table
	redisReq.Key = key
	redisReq.ValueByte = message
	redisReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "HSetByte", redisReq, redisReply, extraInfo)
	if msgErr != nil {
		common.LogError("hSetData Save Redis HSetByte has err", msgErr)
		return msgErr
	}
	return nil
}

func hGetData(table string, key string) ([]byte, *pb.ErrorMessage) {
	redisReq := &pb.RedisMessage{}
	redisReq.Table = table
	redisReq.Key = key
	redisReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "HGetByte", redisReq, redisReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		common.Logger.Error("getMemberInfoFromRedis Call HGetByte has err:", msgErr)
		return nil, msgErr
	}
	return redisReply.ValueByte, nil
}

// DeleteRoomCodeInfo 删除房间码
func hDel(table string, key string) *pb.ErrorMessage {

	roomDelRequest := &pb.RedisMessage{}
	roomDelRequest.Table = table
	roomDelRequest.ValueStringArr = []string{key}
	roomDelReply := &pb.RedisMessage{}
	msgErr := common.Router.Call("Redis", "HDel", roomDelRequest, roomDelReply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		return msgErr
	}
	return nil
}
