package logic

import (
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"math/rand"
	"strconv"
	"time"
)

func init() {
	common.AllComponentMap["Shortid"] = &Shorid{}
}

type Shorid struct {
	base.Base
}

func (self *Shorid) LoadComponent(config *common.OneComponentConfig, componentName string) {
	self.Base.LoadComponent(config, componentName)
}

func (self *Shorid) BeforeStart() {
	genShortIdRequest := &pb.GenShortIdRequest{}
	genShortIdRequest.Length = 6
	genShortIdRequest.RedisKey = common.RedisShortidTable
	_, err := self.Gen(genShortIdRequest, &pb.MessageExtroInfo{})
	if err != nil {
		panic(err)
	}

	// 生成俱乐部的7位邀请码
	genClubInviteCodeRequest := &pb.GenShortIdRequest{}
	genClubInviteCodeRequest.Length = 6
	genClubInviteCodeRequest.RedisKey = common.RedisClubInviteCodeTable
	_, err = self.Gen(genClubInviteCodeRequest, &pb.MessageExtroInfo{})
	if err != nil {
		panic(err)
	}

	// 生成联盟的7位邀请码
	genLeagueInviteCodeRequest := &pb.GenShortIdRequest{}
	genLeagueInviteCodeRequest.Length = 6
	genLeagueInviteCodeRequest.RedisKey = common.RedisLeagueInviteCodeTable
	_, err = self.Gen(genLeagueInviteCodeRequest, &pb.MessageExtroInfo{})
	if err != nil {
		panic(err)
	}
}

/*func (self *Shorid) Start() {
	genShortIdRequest := &pb.GenShortIdRequest{}
	genShortIdRequest.Length = 6
	_, err := self.Gen(genShortIdRequest, &pb.MessageExtroInfo{})
	if err != nil {
		panic(err)
	}
}*/

// 调用:
//genShortIdReq := &pb.GenShortIdRequest{Length: 6}
//genShortIdReply := &pb.GenShortIdReply{}
//extroInfo := &pb.MessageExtroInfo{}
//common.Router.Call("Shortid", "Gen", genShortIdReq, genShortIdReply, extroInfo)
func (self *Shorid) Gen(request *pb.GenShortIdRequest, extroInfo *pb.MessageExtroInfo) (*pb.GenShortIdReply, *pb.ErrorMessage) {
	reply := &pb.GenShortIdReply{}
	rdsReq := &pb.RedisMessage{}
	rdsReq.Table = request.RedisKey
	rdsReply := &pb.RedisMessage{}
	pbErr := common.Router.Call("Redis", "Scard", rdsReq, rdsReply, extroInfo)
	if pbErr != nil {
		common.LogError("Shortid.Gen Call Redis Get has err", pbErr)
		return nil, pbErr
	}

	if rdsReply.Count > 0 {
		return reply, nil
	}

	if false == (request.Length > 0 && request.Length < 10) {
		common.LogError("shortid 的长度必须是 [ 1, 9 ]")
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_GenShortIdFailed, "禁止生成10位长的 shortid")
	}

	minIdx := 1
	for l := 0; l < int(request.Length-1); l++ {
		minIdx *= 10
	}

	maxIdx := minIdx * 10

	for i := minIdx; i < maxIdx; i++ {
		rdsReq.ValueStringArr = append(rdsReq.ValueStringArr, strconv.Itoa(i))
	}

	pbErr = common.Router.Call("Redis", "SAdd", rdsReq, rdsReply, extroInfo)
	if pbErr != nil {
		common.LogError("Shortid.Gen Call Redis Get has err", pbErr)
		return nil, pbErr
	}

	return reply, nil
}

// 从 redis set 中随机取一个 shortid
func (self *Shorid) Get(request *pb.GetShortIdRequest, extroInfo *pb.MessageExtroInfo) (*pb.GetShortIdReply, *pb.ErrorMessage) {
	common.LogDebug("Redis SRANDMEMBER request")

	rdsReq := &pb.RedisMessage{}
	rdsReq.Table = common.RedisShortidTable
	rdsReply := &pb.RedisMessage{}
	pbErr := common.Router.Call("Redis", "Spop", rdsReq, rdsReply, extroInfo)
	if pbErr != nil {
		common.LogError("Shortid.Gen Call Redis Get has err", pbErr)
		return nil, pbErr
	}

	getShortIdReply := &pb.GetShortIdReply{ShortId: rdsReply.ValueString}

	return getShortIdReply, nil
}

// =========================================================
// 俱乐部
// =========================================================

// GetClubInviteCode 生成一个新的俱乐部邀请码
func (c *Shorid) GetClubInviteCode(
	request *pb.GetClubInviteCodeRequest,
	extroInfo *pb.MessageExtroInfo,
) (*pb.GetClubInviteCodeReply, *pb.ErrorMessage) {

	rdsReq := &pb.RedisMessage{}
	rdsReq.Table = common.RedisClubInviteCodeTable
	rdsReply := &pb.RedisMessage{}
	pbErr := common.Router.Call("Redis", "Spop", rdsReq, rdsReply, extroInfo)
	if pbErr != nil {
		common.LogError("Shortid.GetClubInviteCode Call Redis Get has err", pbErr)
		return nil, pbErr
	}

	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(10)
	suffix := strconv.Itoa(i)
	reply := &pb.GetClubInviteCodeReply{InviteCode: rdsReply.ValueString + suffix}

	return reply, nil
}

// =========================================================
// 联盟
// =========================================================

// GetLeagueInviteCode 生成一个新的俱乐部邀请码
func (c *Shorid) GetLeagueInviteCode(
	request *pb.GetLeagueInviteCodeRequest,
	extroInfo *pb.MessageExtroInfo,
) (*pb.GetLeagueInviteCodeReply, *pb.ErrorMessage) {

	rdsReq := &pb.RedisMessage{}
	rdsReq.Table = common.RedisLeagueInviteCodeTable
	rdsReply := &pb.RedisMessage{}
	pbErr := common.Router.Call("Redis", "Spop", rdsReq, rdsReply, extroInfo)
	if pbErr != nil {
		common.LogError("Shortid.GetLeagueInviteCode Call Redis Get has err", pbErr)
		return nil, pbErr
	}

	rand.Seed(time.Now().UnixNano())
	i := rand.Intn(10)
	suffix := strconv.Itoa(i)
	reply := &pb.GetLeagueInviteCodeReply{InviteCode: rdsReply.ValueString + suffix}

	return reply, nil
}
