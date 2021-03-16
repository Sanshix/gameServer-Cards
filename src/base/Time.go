package base

import (
	"gameServer-demo/src/common"
	"strconv"
	"time"
)

func init() {
	common.AllComponentMap["Time"] = &Time{}
}

// Time 时间组件，有一些时区修正功能
type Time struct {
	Base
	common.TimeI
	// 当前依赖时区
	timeLocation *time.Location
}

//LoadComponent 加载组件
func (t *Time) LoadComponent(config *common.OneComponentConfig, componentName string) {
	t.Base.LoadComponent(config, componentName)
	offset, err := strconv.Atoi((*t.Config)["TimeZoneOffset"])
	if err != nil {
		panic(err)
	}
	timeLocation := time.FixedZone((*t.Config)["TimeZoneName"], offset*3600)
	t.timeLocation = timeLocation
}

//Start 开启组件
func (t *Time) Start() {
}

// GetTimeByTimeStamp 通过时间戳获取一个时间结构体
func (t *Time) GetTimeByTimeStamp(timeStamp int64) time.Time {
	return time.Unix(timeStamp, 0).In(t.timeLocation)
}

// GetNowWeekTimeStr 获取一个周时间字符串 年-周数-周几 组合的时间字段
func (t *Time) GetNowWeekTimeStr() string {
	nowTime := t.GetTimeByTimeStamp(time.Now().Unix())
	nowYear, nowWeek := nowTime.ISOWeek()
	nowWeekDay := int(nowTime.Weekday())
	if nowWeekDay == 0 {
		nowWeekDay = 7
	}
	return strconv.Itoa(nowYear) + "-" + strconv.Itoa(nowWeek) + "-" + strconv.Itoa(nowWeekDay)
}

// IsSameDay 是否是同一天
// 参数：两个需要比较的时间戳，跨天判定点的时和分
// 如跨天点被界定在凌晨0点，则hour传0，min传0
// 跨天点被界定在凌晨4点，则hour传4，min传0
func (t *Time) IsSameDay(timeStamp1 int64, timeStamp2 int64, hour int, min int) bool {
	if timeStamp1 == timeStamp2 {
		return true
	}
	if timeStamp1 > timeStamp2 {
		temp := timeStamp1
		timeStamp1 = timeStamp2
		timeStamp2 = temp
	}
	time1 := t.GetTimeByTimeStamp(timeStamp1)
	time2 := t.GetTimeByTimeStamp(timeStamp2)
	year1, month1, day1 := time1.Date()
	year2, month2, day2 := time2.Date()
	if year1 != year2 || month1 != month2 || day1 != day2 {
		return false
	}
	hour1, min1, _ := time1.Clock()
	hour2, min2, _ := time2.Clock()
	if hour1 < hour && hour2 > hour {
		return false
	}
	if hour1 == hour && hour2 == hour {
		if min1 < min && min2 > min {
			return false
		}
	}
	return true
}

// IsSameWeek 是否是同一周
// 参数：两个需要比较的时间戳
func (t *Time) IsSameWeek(timeStamp1 int64, timeStamp2 int64) bool {
	if timeStamp1 == timeStamp2 {
		return true
	}
	if timeStamp1 > timeStamp2 {
		temp := timeStamp1
		timeStamp1 = timeStamp2
		timeStamp2 = temp
	}
	time1 := t.GetTimeByTimeStamp(timeStamp1)
	time2 := t.GetTimeByTimeStamp(timeStamp2)

	year1, week1 := time1.ISOWeek()
	year2, week2 := time2.ISOWeek()
	if year1 == year2 && week1 == week2 {
		return true
	}
	return false
}

// GetTimeStr 获取时间戳所属的周的字符串和所属天的字符串
// 返回：所属周的字符串，所属天的字符串（用第几周的周几代替）
func (t *Time) GetTimeStr(timeStamp int64) (string, string) {
	nowTime := t.GetTimeByTimeStamp(timeStamp)
	year, week := nowTime.ISOWeek()
	weekDay := int(nowTime.Weekday())
	if weekDay == 0 {
		weekDay = 7
	}
	weekStr := strconv.Itoa(year) + "-" + strconv.Itoa(week)
	dayStr := weekStr + "-" + strconv.Itoa(weekDay)
	return weekStr, dayStr
}

func (a *Time) GetDate(t time.Time) int64 {
	timeStr := t.Format("2006-01-02")
	parseTime, _ := time.Parse("2006-01-02", timeStr)
	return parseTime.Unix()
}

func (a *Time) ParseWeekToDate(year int, isoWeek int, weekDay time.Weekday) int64 {
	date := time.Date(year, 0, 0, 0, 0, 0, 0, a.timeLocation)
	isoYear, isoWeek := date.ISOWeek()
	for date.Weekday() != time.Monday { // iterate back to Monday
		date = date.AddDate(0, 0, -1)
		isoYear, isoWeek = date.ISOWeek()
	}
	for isoYear < year { // iterate forward to the first day of the first week
		date = date.AddDate(0, 0, 1)
		isoYear, isoWeek = date.ISOWeek()
	}
	for isoWeek < isoWeek { // iterate forward to the first day of the given week
		date = date.AddDate(0, 0, 1)
		isoYear, isoWeek = date.ISOWeek()
	}
	date = date.AddDate(0, 0, int(weekDay))
	return date.Unix()
}
