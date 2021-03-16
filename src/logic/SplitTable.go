package logic

import (
	"database/sql"
	"fmt"
	"gameServer-demo/src/base"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"sort"
	"strconv"
	"strings"
	"time"
)

func init() {
	common.AllComponentMap["SplitTable"] = &SplitTable{}
}

type SplitTable struct {
	base.Base
	db *sql.DB
	//分表粒度（天）
	splitValue           int
	splitTableEndTimes   map[string]int64
	createSplitTableSqls map[string]string
	splitTableRecords    map[string][]*SplitTableRecord
}

//分表记录
type SplitTableRecord struct {
	autoId           int32
	tableName        string
	splitName        string
	startTimestamp   int64
	endTimestamp     int64
	createdTimestamp int64
}

// LoadComponent 加载组件
func (obj *SplitTable) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)

	dbTemp, err := sql.Open("mysql", (*obj.Config)["connect_string"])
	if err != nil {
		common.LogError("Configer connect mysql err", err)
		panic(err)
	}
	dbTemp.SetConnMaxLifetime(600 * time.Second)
	dbTemp.SetMaxIdleConns(0)
	//dbTemp.SetMaxOpenConns(30)
	obj.db = dbTemp

	//解析分表粒度（单位：天）
	splitValueStr, exists := (*obj.Config)["split_value"]
	if !exists {
		obj.splitValue = 7
	} else {
		splitValue, convertErr := strconv.Atoi(splitValueStr)
		if convertErr != nil {
			obj.splitValue = 7
		} else {
			obj.splitValue = splitValue
		}

	}

	obj.initSplitTableMap()
	return
}

// Start 这个方法将在所有组件的LoadComponent之后依次调用
func (obj *SplitTable) Start() {
	obj.Base.Start()
	obj.checkTable()
	obj.initSplitTableRecord()
	obj.createSplitTableTask()
}

//初始化需要分表的表
func (obj *SplitTable) initSplitTableMap() {
	obj.splitTableEndTimes = make(map[string]int64)
	obj.createSplitTableSqls = make(map[string]string)
	obj.splitTableRecords = make(map[string][]*SplitTableRecord)

	obj.createSplitTableSqls[common.MysqlReportGameRecord] = common.MysqlCheckReportGameRecord
	obj.createSplitTableSqls[common.MysqlReportBalanceChangeRecord] = common.MysqlCheckReportBalanceChangeRecord
	obj.createSplitTableSqls[common.MysqlReportBonusRecord] = common.MysqlCheckReportBonusRecord

	//TODO 在此处增加需要分表的表和创建表的sql
}

//检测分表记录表
func (obj *SplitTable) checkTable() {
	_, err := obj.db.Exec(common.MysqlCheckConfigSplitTableInfo)
	if err != nil {
		common.LogError("Mysql checkTable MysqlConfigSplitTableInfo has err", err)
		return
	}
}

func (obj *SplitTable) initSplitTableRecord() {
	rows, err := obj.db.Query("select auto_id,table_name,split_name,start_timestamp,end_timestamp,created_timestamp from " + common.MysqlConfigSplitTableInfo)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select split table record from "+common.MysqlConfigSplitTableInfo+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		var out = &SplitTableRecord{}
		err = rows.Scan(
			&out.autoId,
			&out.tableName,
			&out.splitName,
			&out.startTimestamp,
			&out.endTimestamp,
			&out.createdTimestamp)
		if err != nil {
			common.LogError("select split table record from "+common.MysqlConfigGlobal+" has err", err)
			continue
		}
		arr, exists := obj.splitTableRecords[out.tableName]
		if !exists {
			arr = make([]*SplitTableRecord, 0)
		}
		arr = append(arr, out)
		obj.splitTableRecords[out.tableName] = arr
		endTime, exists := obj.splitTableEndTimes[out.tableName]
		if !exists {
			endTime = 0
		}
		if out.endTimestamp > endTime {
			obj.splitTableEndTimes[out.tableName] = out.endTimestamp
		}
	}
}

//创建分表
func (obj *SplitTable) createSplitTable(tableName string, suffix string, startTimestamp int64, endTimestamp int64) {
	common.LogDebug("create new table:", tableName, suffix, startTimestamp, endTimestamp)
	createTableSql, exits := obj.createSplitTableSqls[tableName]
	if !exits {
		common.LogInfo("can not find create table sql", tableName, suffix)
		return
	}
	//替换表名
	realTableName := tableName + "_" + suffix

	realCreateSql := strings.Replace(createTableSql, tableName, realTableName, 1)
	_, err := obj.db.Exec(realCreateSql)
	if err != nil {
		common.LogError("Mysql checkTable has err", realTableName, err)
		return
	}
	now := time.Now().Unix()
	//写入记录
	_, err = obj.db.Exec(
		`INSERT INTO `+common.MysqlConfigSplitTableInfo+` (
			table_name,split_name,start_timestamp,end_timestamp,created_timestamp)
			VALUES (?,?,?,?,?)`,
		tableName,
		realTableName,
		startTimestamp,
		endTimestamp,
		now)
	if err != nil {
		common.LogError("insert split table record err:", realTableName, err)
		return
	}

	//写入本地缓存
	cache := &SplitTableRecord{}
	cache.tableName = tableName
	cache.splitName = realTableName
	cache.startTimestamp = startTimestamp
	cache.endTimestamp = endTimestamp
	cache.createdTimestamp = now
	arr, exists := obj.splitTableRecords[cache.tableName]
	if !exists {
		arr = make([]*SplitTableRecord, 0)
	}
	common.LogDebug("begin insert table name cache:", cache, arr)

	arr = append(arr, cache)
	obj.splitTableRecords[cache.tableName] = arr
	endTime, exists := obj.splitTableEndTimes[cache.tableName]
	if !exists {
		endTime = 0
	}
	if cache.endTimestamp > endTime {
		obj.splitTableEndTimes[cache.tableName] = cache.endTimestamp
	}
	common.LogDebug("end insert table name cache:", cache, arr)
}

//创建分表任务
func (obj *SplitTable) createSplitTableTask() {
	//5分钟执行一次检测
	common.StartTimer(time.Duration(5)*time.Minute, true, func() bool {
		//比较2小时后
		compareTime := time.Now().Add(time.Duration(2) * time.Hour).Unix()
		for key, _ := range obj.createSplitTableSqls {
			startTime, exits := obj.splitTableEndTimes[key]
			if !exits {
				startTime = obj.getZeroTime(time.Now()).Unix()
			}
			//如果结束时间大于或等于2小时后，也就是提前2小时创建下1个分表
			if startTime <= compareTime || !exits {
				tm := time.Unix(startTime, 0)
				endTimestamp := tm.Add(time.Duration(24*obj.splitValue) * time.Hour).Unix()
				suffix := fmt.Sprintf("%d%s%s", tm.Year(), obj.parseNum2Str(int(tm.Month())), obj.parseNum2Str(tm.Day()))
				obj.createSplitTable(key, suffix, startTime, endTimestamp)
			}
		}
		return true
	})
}

func (obj *SplitTable) parseNum2Str(num int) string {
	if num < 10 {
		return "0" + strconv.Itoa(num)
	}
	return strconv.Itoa(num)
}

//获取某一天的0点时间
func (obj *SplitTable) getZeroTime(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
}

// GetSplitTableNames 获取分表列表
func (obj *SplitTable) GetSplitTableNames(request *pb.GetSplitTablesRequest, extroInfo *pb.MessageExtroInfo) (*pb.GetSplitTablesReply, *pb.ErrorMessage) {
	records, exists := obj.splitTableRecords[request.TableName]
	reply := &pb.GetSplitTablesReply{}
	reply.TableNames = make([]string, 0)
	if !exists {
		reply.TableNames = append(reply.TableNames, request.TableName)
		return reply, nil
	}

	var stringSlice sort.StringSlice
	for i := range records {
		current := records[i]
		if (request.StartTimestamp <= current.startTimestamp && request.EndTimestamp >= current.startTimestamp) ||
			(request.StartTimestamp >= current.startTimestamp && request.EndTimestamp < current.endTimestamp) ||
			(request.StartTimestamp < current.endTimestamp && request.EndTimestamp >= current.endTimestamp) {
			//reply.TableNames = append(reply.TableNames, current.splitName)
			stringSlice = append(stringSlice, current.splitName)
		}
	}
	//common.LogDebug("GetSplitTableNames records:", records, request.TableName, request.StartTimestamp, request.EndTimestamp, reply.TableNames)

	sort.Sort(stringSlice)
	for _, item := range stringSlice {
		reply.TableNames = append(reply.TableNames, item)
	}

	return reply, nil

}
