package common

import (
	"bytes"
	pb "gameServer-demo/src/grpc"
	"strconv"
)

// CreateSmsCaptcha 创建短信验证码
// mobileNum 手机号
// exTime 验证码的有效时长，秒为单位
// intervalTime 验证码的生成间隔，秒为单位
func CreateSmsCaptcha(mobileNum string, exTime int32, intervalTime int32) (string, *pb.ErrorMessage) {
	captchaTemp := ""
	request := &pb.RedisMessage{
		Table: RedisSmsCaptchaIntervalTimeTable,
		Key:   mobileNum,
	}
	extra := &pb.MessageExtroInfo{}
	reply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "GetString", request, reply, extra)
	if msgErr != nil {
		LogError("CreateSmsCaptcha call Redis GetString has error：", msgErr)
		return captchaTemp, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	if reply.GetValueString() != "" {
		LogError("CreateSmsCaptcha TooFrequentOperation", mobileNum, msgErr)
		return captchaTemp, GetGrpcErrorMessage(pb.ErrorCode_TooFrequentOperation, "")
	}

	var buffer bytes.Buffer

	for index := 0; index < 6; index++ {
		randomNum := GetRandomNum(0, 9)
		buffer.WriteString(strconv.Itoa(randomNum))
	}
	captchaTemp = buffer.String()
	request = &pb.RedisMessage{
		Table:       RedisSmsCaptchaTable,
		Key:         mobileNum,
		ExTime:      exTime,
		ValueString: captchaTemp,
	}
	reply = &pb.RedisMessage{}
	msgErr = Router.Call("Redis", "SetString", request, reply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("CreateSmsCaptcha call Redis SetString has error：", msgErr)
		return captchaTemp, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	request = &pb.RedisMessage{
		Table:       RedisSmsCaptchaIntervalTimeTable,
		Key:         mobileNum,
		ExTime:      intervalTime,
		ValueString: "",
	}
	reply = &pb.RedisMessage{}
	msgErr = Router.Call("Redis", "SetString", request, reply, &pb.MessageExtroInfo{})
	if msgErr != nil {
		LogError("CreateSmsCaptcha call Redis SetString RedisSmsCaptchaIntervalTimeTable has error：", msgErr)
		return captchaTemp, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return captchaTemp, nil
}

// VerifyCaptcha 验证短信验证码
// mobileNum 手机号
// captcha 验证码
// 返回是否通过验证
func VerifyCaptcha(mobileNum string, captcha string) bool {
	request := &pb.RedisMessage{
		Table: RedisSmsCaptchaTable,
		Key:   mobileNum,
	}
	extra := &pb.MessageExtroInfo{}
	reply := &pb.RedisMessage{}
	msgErr := Router.Call("Redis", "GetString", request, reply, extra)
	if msgErr != nil {
		LogError("VerifyCaptcha call Redis GetString has error：", msgErr)
		return false
	}
	if reply.GetValueString() == "" {
		return false
	}
	if reply.GetValueString() != captcha {
		return false
	}
	return true
}
