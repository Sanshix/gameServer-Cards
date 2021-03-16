package common

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	pb "gameServer-demo/src/grpc"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// LogInfo 输出info类型的日志
func LogInfo(a ...interface{}) {
	if Logger != nil {
		Logger.Info(a...)
	} else {
		fmt.Println(a...)
	}
}

// LogError 输出error类型的日志
func LogError(a ...interface{}) {
	if Logger != nil {
		Logger.Error(a...)
	} else {
		fmt.Println(a...)
	}
}

// LogDebug 输出info类型的日志，在logger组件配置中可以选则是否输出debug信息
func LogDebug(a ...interface{}) {
	if Logger != nil {
		Logger.Debug(a...)
	} else {
		fmt.Println(a...)
	}
}

// GetGrpcErrorMessage 通过传入参数获得一个rpc错误信息
func GetGrpcErrorMessage(code pb.ErrorCode, message string) *pb.ErrorMessage {
	errMessage := &pb.ErrorMessage{}
	errMessage.Code = code
	errMessage.Message = message
	return errMessage
}

// TimerCallBack 定时器回掉函数
type TimerCallBack func() bool

// StartTimer 开启定时器
//
// 参数：
//
// d：定时器的间隔时间
//
// atOnce：是否会被立即执行一次回掉，为true的话，回立马执行一次回掉，然后在指定时间之后继续执行回掉
//
// callBack：定时器回掉，在d时间之后执行此函数，这个函数返回true则会不断执行，返回false则不再继续执行回掉
//
// 返回值：返回定时器对象和停止通道，以供StopTimer使用，完整的停止一个定时器请调用文件中的StopTimer
func StartTimer(d time.Duration, atOnce bool, callBack TimerCallBack) (*time.Timer, chan bool) {
	if atOnce {
		needContinue := callBack()
		if !needContinue {
			return nil, nil
		}
	}
	tm := time.NewTimer(d)
	stopChan := make(chan bool, 1)
	go func() {
		for {
			select {
			case <-tm.C:
				needContinue := callBack()
				if !needContinue {
					tm.Stop()
					//close(stopChan)
					return
				}
				tm.Reset(d)
			case stop := <-stopChan:
				if stop {
					return
				}
			}

		}
	}()

	return tm, stopChan
}

// StopTimer 停止定时器
//
// 参数：
//
// timer：需要停止的定时器对象
//
// stopChan：对应定时器对象的停止通道，以确保定时器协程完全退出
func StopTimer(timer *time.Timer, stopChan chan bool) {
	if timer != nil {
		timer.Stop()
	}
	if stopChan != nil {
		stopChan <- true
		close(stopChan)
	}
}

// IndexOf 查找数组中是否包含一个元素
// array是任意类型的数组
// elem是判断是否包含此元素
func IndexOf(array interface{}, elem interface{}) int {
	sliceLen := reflect.ValueOf(array).Len()
	realArray := make([]interface{}, sliceLen)
	for i := 0; i < sliceLen; i++ {
		realArray[i] = reflect.ValueOf(array).Index(i).Interface()
	}

	for index, oneMember := range realArray {
		if reflect.TypeOf(oneMember).Kind() != reflect.TypeOf(elem).Kind() {
			return -1
		}
		if reflect.DeepEqual(oneMember, elem) {
			return index
		}
	}
	return -1
}
func IndexOfByFunc(array interface{}, compareFunc func(current interface{}) bool) int {
	sliceLen := reflect.ValueOf(array).Len()
	realArray := make([]interface{}, sliceLen)
	for i := 0; i < sliceLen; i++ {
		realArray[i] = reflect.ValueOf(array).Index(i).Interface()
	}
	for index, oneMember := range realArray {
		if compareFunc(oneMember) {
			return index
		}
	}
	return -1
}

// GetRandomNum 获得一个范围内的随机数
func GetRandomNum(min int, max int) int {
	if min == max {
		return min
	}
	num := rand.Intn(max-min+1) + min
	return num
}

// GenerateRandomNumber 生成count个[start,end)结束的不重复的随机数
func GenerateRandomNumber(start int, end int, count int) []int {
	//范围检查
	if end < start || (end-start) < count {
		return nil
	}

	//存放结果的slice
	nums := make([]int, 0)
	for len(nums) < count {
		//生成随机数
		num := rand.Intn(end-start) + start

		//查重
		exist := false
		for _, v := range nums {
			if v == num {
				exist = true
				break
			}
		}

		if !exist {
			nums = append(nums, num)
		}
	}

	return nums
}

// Md5 加密
func Md5(orgStr string) string {
	h := md5.New()
	h.Write([]byte(orgStr))
	return hex.EncodeToString(h.Sum(nil))
}

// EncodePassword 加密密码
func EncodePassword(password string) string {
	passwordSalt := "1IyAyYK1F@smzEdZpSGQF!%C&DCHZWSQ"
	return Md5(password + passwordSalt)
}

// ValidatePassword 验证密码
func ValidatePassword(password string, encryptedPassword string) bool {
	return EncodePassword(password) == encryptedPassword
}

// ByteArrayToMsg 将byte数组转换为消息字符串，不是转换为普通的字符串
func ByteArrayToMsg(b []byte) string {
	messageLen := len(b)
	var buffer bytes.Buffer
	strMsg := ""
	for index := 0; index < messageLen; index++ {
		//strMsg = strMsg + strconv.Itoa(int(b[index]))
		buffer.WriteString(strconv.Itoa(int(b[index])))
		if index != messageLen-1 {
			//strMsg = strMsg + ","
			buffer.WriteString(",")
		}
	}
	strMsg = buffer.String()
	return strMsg
}

// MsgToByteArray 将消息字符串转换为byte数组，不是普通的字符串
func MsgToByteArray(m string) []byte {
	s := strings.Split(m, ",")
	messageLen := len(s)
	realMessage := make([]byte, messageLen)
	for index := 0; index < messageLen; index++ {
		msgInt, err := strconv.Atoi(s[index])
		if err != nil {
			break
		}
		realMessage[index] = byte(msgInt)
	}
	return realMessage
}

// RandSlice 切片乱序
func RandSlice(slice interface{}) {
	rv := reflect.ValueOf(slice)
	if rv.Type().Kind() != reflect.Slice {
		return
	}

	length := rv.Len()
	if length < 2 {
		return
	}

	swap := reflect.Swapper(slice)
	rand.Seed(time.Now().Unix())
	for i := length - 1; i >= 0; i-- {
		j := rand.Intn(length)
		swap(i, j)
	}
	return
}

// GetRandomIndexByWeight 通过权重数组，获得一个随机值，值是权重数组的索引
func GetRandomIndexByWeight(weight interface{}) (int, error) {
	realWeight := []int{}
	switch weight.(type) {
	case []int:
		weightChange, ok := weight.([]int)
		if !ok {
			LogError("GetRandomIndexByWeight weight is not []int")
			return 0, errors.New("weight is not []int")
		}
		realWeight = weightChange
	case []int32:
		weightChange, ok := weight.([]int32)
		if !ok {
			LogError("GetRandomIndexByWeight weight is not []int")
			return 0, errors.New("weight is not []int")
		}
		for _, oneInt32 := range weightChange {
			realWeight = append(realWeight, int(oneInt32))
		}
	case []string:
		weightChange, ok := weight.([]string)
		if !ok {
			LogError("GetRandomIndexByWeight weight is not []string")
			return 0, errors.New("weight is not []string")
		}
		for _, oneStr := range weightChange {
			oneNum, err := strconv.Atoi(oneStr)
			if err != nil {
				LogError("GetRandomIndexByWeight strconv.Atoi(oneStr) has err", err)
				return 0, err
			}
			realWeight = append(realWeight, oneNum)
		}
	default:
		LogError("GetRandomIndexByWeight weight type has err")
		return 0, errors.New("weight type error")
	}
	totalWeight := 0
	for _, oneWeight := range realWeight {
		totalWeight += oneWeight
	}
	randomNum := GetRandomNum(1, totalWeight)
	totalWeight = 0
	for index, oneWeight := range realWeight {
		totalWeight += oneWeight
		if randomNum <= totalWeight {
			return index, nil
		}
	}
	LogError("GetRandomIndexByWeight weight can not select", weight)
	return 0, errors.New("weight can not select")
}

//SortDescByWinOrLose 房间玩家按照输赢排序，返回有序切片，不影响玩家原来的房间座次
func SortDescByWinOrLose(userList []*pb.RoomPlayerInfo) []*pb.RoomPlayerInfo {
	winner := make([]*pb.RoomPlayerInfo, 0)
	for _, user := range userList {
		if user.GetPlayerRoomState() == pb.PlayerRoomState_PlayerRoomStatePlay {
			winner = append(winner, user)
		}

	}
	swap := reflect.Swapper(winner)
	for i := 0; i < len(winner); i++ {
		for j := i + 1; j < len(winner); j++ {
			if winner[i].WinOrLose < winner[j].WinOrLose {
				swap(i, j)
			}
		}
	}
	return winner
}

// 获取int64绝对值
func AbsInt64(n int64) int64 {
	y := n >> 63       // y ← x >> 63
	return (n ^ y) - y // (x ⨁ y) - y
}

// 查询服务器 某组件 是否存在
// 参数：componentName组件名
func SelectComponentExist(componentName string) bool {
	for k, _ := range ServerConfig {
		if _, ok := ServerConfig[k][componentName]; ok {
			return true
		}
	}
	return false
}

func ParseRoomPlayer2PlayerInfo(roomPlayer *pb.RoomPlayerInfo) *pb.PlayerInfo {
	return &pb.PlayerInfo{
		Uuid:       roomPlayer.Uuid,
		ShortId:    roomPlayer.ShortId,
		Name:       roomPlayer.Name,
		Account:    roomPlayer.Account,
		Balance:    roomPlayer.Balance,
		IsRobot:    roomPlayer.IsRobot,
		Role:       roomPlayer.Role,
		Sex:        roomPlayer.Sex,
		HeadImgUrl: roomPlayer.HeadImgUrl,
		Country:    roomPlayer.Country,
		Province:   roomPlayer.Province,
		City:       roomPlayer.City,
	}
}

// 将以逗号分隔的字符串数字转换为数组
// 请满足格式
// eg: "1,3,4,6" -> [1,3,4,6]
func TakeStringsToIntArr(a string) []int {
	reply := make([]int, 0)
	strs := strings.Split(a, ",")
	for _, v := range strs {
		realInt, err := strconv.Atoi(v)
		if err != nil {
			LogError("function TakeStringsToIntArr Atoi has err：", err)
			return nil
		}
		reply = append(reply, realInt)
	}
	return reply
}

// 检测是否开启了某模式
func CheckModeOpen(mode pb.GameMode) bool {
	for _, v := range SubMode {
		if v == mode {
			return true
		}
	}
	return false
}

// 查询某string是否在[]string里面
func SelectStringInArray(str string, arr []string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}

// 一个比值在范围内是否触发
// eg: 1,100,20 返回有20% true 80%false
func GetRandIsTouch(min, max, ratio int) bool {
	if GetRandomNum(min, max) >= ratio {
		return true
	}
	return false
}

// TwoArr2OneArr 二维坐标转换为一维坐标
func TwoArr2OneArr(rowNum int, columnNum int, x int, y int) (int, error) {
	if x >= columnNum || x < 0 {
		return 0, errors.New("x >= columnNum || x < 0")
	}
	if y >= rowNum || y < 0 {
		return 0, errors.New("y >= rowNum || y < 0")
	}
	return y*columnNum + x, nil
}

// OneArr2TwoArr 一维坐标转换为二维坐标
func OneArr2TwoArr(rowNum int, columnNum int, index int) (int, int, error) {
	x := index % columnNum
	y := index / columnNum
	return x, y, nil
}
