/*
	用于redis操作
*/

package base

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"time"

	"github.com/gomodule/redigo/redis"
)

func init() {
	common.AllComponentMap["Redis"] = &Redis{}
}

// Redis 组件提供Redis的一些常用接口，都是rpc接口
type Redis struct {
	common.RedisI
	Base
	Pool redis.Pool
}

// LoadComponent 加载组件
func (obj *Redis) LoadComponent(config *common.OneComponentConfig, componentName string) {
	obj.Base.LoadComponent(config, componentName)
	obj.Pool = redis.Pool{
		MaxIdle:     16,
		IdleTimeout: 180 * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", (*obj.Config)["redis_host"])
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
	return
}

// SetByte 函数调用redis的set方法，存入byte数据
func (obj *Redis) SetByte(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis SetByte request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	value := request.GetValueByte()
	reply := &pb.RedisMessage{}
	realKey := table + ":" + key
	_, err := conn.Do("SET", realKey, value)
	if err != nil {
		common.LogError("Redis SetByte has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// SetString 函数调用redis的set方法，存入string数据
func (obj *Redis) SetString(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis SetString request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	reply := &pb.RedisMessage{}
	value := request.GetValueString()
	exTime := request.GetExTime()

	realKey := table + ":" + key
	if exTime > 0 {
		_, err := conn.Do("SETEX", realKey, exTime, value)
		if err != nil {
			common.LogError("Redis SetString has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
	} else {
		_, err := conn.Do("SET", realKey, value)
		if err != nil {
			common.LogError("Redis SetString has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
	}

	return reply, nil
}

// GetByte 函数调用redis的get方法，读取byte数据
func (obj *Redis) GetByte(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis GetByte request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	reply := &pb.RedisMessage{}
	realKey := table + ":" + key
	reply.ValueByte = nil
	res, err := redis.Bytes(conn.Do("GET", realKey))
	if err == redis.ErrNil {
		return reply, nil
	}
	if err != nil {
		common.LogError("Redis GetByte has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueByte = res
	return reply, nil
}

// GetString 函数调用redis的get方法，读取string数据
func (obj *Redis) GetString(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis GetString request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	reply := &pb.RedisMessage{}
	realKey := table + ":" + key
	reply.ValueString = ""
	res, err := redis.String(conn.Do("GET", realKey))
	if err == redis.ErrNil {
		return reply, nil
	}
	if err != nil {
		common.LogError("Redis GetString has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueString = res
	return reply, nil
}

// Delete 函数调用redis的del方法，删除一个主键数据
func (obj *Redis) Delete(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis Delete request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	reply := &pb.RedisMessage{}
	realKey := table + ":" + key
	_, err := conn.Do("DEL", realKey)
	if err != nil {
		common.LogError("Redis Delete has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// SRandMember 函数调用redis的SRandMember方法，获得一个集合里固定数量的元素
func (obj *Redis) SRandMember(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis SRandMember request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	count := request.GetCount()
	reply := &pb.RedisMessage{}
	realKey := table + ":" + key
	res, err := redis.Strings(conn.Do("SRANDMEMBER", realKey, count))
	if err != nil {
		common.LogError("Redis SRandMember has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueStringArr = res
	return reply, nil
}

// SIsMember 函数调用redis的SIsMember方法，检测一个元素是否在集合里
func (obj *Redis) SIsMember(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis SIsMember request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	member := request.GetValueString()
	reply := &pb.RedisMessage{}
	realKey := table + ":" + key
	res, err := redis.Bool(conn.Do("SISMEMBER", realKey, member))
	if err != nil {
		common.LogError("Redis SIsMember has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueBool = res
	return reply, nil
}

// SRem 函数调用redis的SRem方法，从集合中删除一批元素
func (obj *Redis) SRem(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis SRem request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	members := request.GetValueStringArr()
	reply := &pb.RedisMessage{}
	realKey := table + ":" + key
	_, err := conn.Do("SREM", realKey, members)
	if err != nil {
		common.LogError("Redis SRem has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// SAdd 函数调用redis的SAdd方法，向集合中添加一批元素
func (obj *Redis) SAdd(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis SAdd request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	members := request.GetValueStringArr()

	realKey := table + ":" + key
	rest := make([]interface{}, len(members)+1)
	rest[0] = realKey
	for key, el := range members {
		rest[key+1] = el
	}

	reply := &pb.RedisMessage{}
	_, err := conn.Do("SADD", rest...)
	if err != nil {
		common.LogError("Redis SAdd has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// Scard 函数调用redis的Scard方法，查询集合中元素的数量
func (obj *Redis) Scard(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis Scard request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()

	realKey := table + ":" + key
	reply := &pb.RedisMessage{}
	cnt, err := redis.Int(conn.Do("SCARD", realKey))
	if err != nil {
		common.LogError("Redis SCARD has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.Count = int64(cnt)

	return reply, nil
}

// SMEMBERS 获取set中全部值
func (obj *Redis) SMembers(request *pb.RedisMessage, extraInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()

	realKey := table + ":" + key
	reply := &pb.RedisMessage{}
	ret, err := redis.Strings(conn.Do("SMEMBERS", realKey))
	if err != nil {
		common.LogError("Redis SMEMBERS has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueStringArr = ret
	return reply, nil
}

// Spop 函数调用redis的Spop方法，移除并返回集合中的一个随机元素
func (obj *Redis) Spop(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis Spop request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()

	realKey := table + ":" + key
	reply := &pb.RedisMessage{}
	ret, err := redis.String(conn.Do("SPOP", realKey))
	if err != nil {
		common.LogError("Redis SPOP has err", err, realKey)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueString = ret

	return reply, nil
}

//根据命令获取Set中的交(SINTER)/并(SUNION)/差集(SDIFF)
func (obj *Redis) SMembersByCode(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	conn := obj.Pool.Get()
	defer conn.Close()
	reply := &pb.RedisMessage{}
	ret, err := redis.String(conn.Do(request.Key, request.ValueStringArr))
	if err != nil {
		common.LogError("Redis SMembersByCode has err", err, request)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueString = ret

	return reply, nil
}

// HKeys 函数调用redis的HKeys方法，从哈希表中取得所有key的值
func (obj *Redis) HKeys(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis HKeys request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	reply := &pb.RedisMessage{}
	ret, err := redis.Strings(conn.Do("HKEYS", table))
	if err != nil {
		common.LogError("Redis HKeys has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueStringArr = ret
	return reply, nil
}

// HSetByte 函数调用redis的Hset方法，向哈希表中设置一个byte值
func (obj *Redis) HSetByte(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis HSetByte request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	value := request.GetValueByte()

	reply := &pb.RedisMessage{}
	_, err := conn.Do("HSET", table, key, value)
	if err != nil {
		common.LogError("Redis HSetByte has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// HGetByte 函数调用redis的Hget方法，从哈希表中取一个byte值
func (obj *Redis) HGetByte(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis HGetByte request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()

	reply := &pb.RedisMessage{}
	ret, err := redis.Bytes(conn.Do("HGET", table, key))
	if err == redis.ErrNil {
		return reply, nil
	}
	if err != nil {
		common.LogError("Redis HGetByte has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueByte = ret
	return reply, nil
}

// HSetString 函数调用redis的Hset方法，向哈希表中设置一个string值
func (obj *Redis) HSetString(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis HSetString request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()
	value := request.GetValueString()

	reply := &pb.RedisMessage{}
	_, err := conn.Do("HSET", table, key, value)
	if err != nil {
		common.LogError("Redis HSetString has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// HGetString 函数调用redis的Hget方法，从哈希表中取一个string值
func (obj *Redis) HGetString(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis HGetByte request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	key := request.GetKey()

	reply := &pb.RedisMessage{}
	ret, err := redis.String(conn.Do("HGET", table, key))
	if err == redis.ErrNil {
		return reply, nil
	}
	if err != nil {
		common.LogError("Redis HGetString has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueString = ret
	return reply, nil
}

// HDel 函数调用redis的Hdel方法，从哈希表中删除一个key
func (obj *Redis) HDel(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis HDel request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	delKeys := request.GetValueStringArr()

	rest := make([]interface{}, len(delKeys)+1)
	rest[0] = table
	for key, el := range delKeys {
		rest[key+1] = el
	}
	reply := &pb.RedisMessage{}
	_, err := conn.Do("HDEL", rest...)
	if err != nil {
		common.LogError("Redis HDel has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// LPushByteWithCount 函数调用redis的LPush和LTrim方法，在列表表头插入多个byte元素,并且将列表元素个数限制在count个
func (obj *Redis) LPushByteWithCount(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis HDel request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	count := request.GetCount()
	pushInfos := request.GetValueByteArr()

	rest := make([]interface{}, len(pushInfos)+1)
	rest[0] = table
	for key, el := range pushInfos {
		rest[key+1] = el
	}
	count = count - 1
	if count <= 0 {
		count = 1
	}
	reply := &pb.RedisMessage{}
	conn.Send("MULTI")
	conn.Send("LPUSH", rest...)
	conn.Send("LTRIM", table, 0, count)
	_, err := conn.Do("EXEC")
	if err != nil {
		common.LogError("Redis LPushByteWithCount redis has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// LRangeByte 函数调用redis的LRSNGE方法，获取指定数量的byte元素
func (obj *Redis) LRangeByte(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//common.LogDebug("Redis HDel request", request)
	conn := obj.Pool.Get()
	defer conn.Close()
	table := request.GetTable()
	start := request.GetStart()
	stop := request.GetStop()

	reply := &pb.RedisMessage{}
	byteSlices, err := redis.ByteSlices(conn.Do("LRANGE", table, start, stop))
	if err != nil {
		common.LogError("Redis LRangeByte ByteSlices has error", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueByteArr = byteSlices
	return reply, nil
}

// ZAddStringArr 调用redis的zAdd方法，
// request参数: table名,key,ValueStringArr
// 返回：无
func (obj *Redis) ZAddStringArr(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//获取表名，key为时间，存储的信息
	table := request.GetTable()
	key := request.GetKey()
	realTable := table + ":" + key
	ValueStringArr := request.GetValueStringArr()

	reply := &pb.RedisMessage{}
	conn := obj.Pool.Get()
	defer conn.Close()
	//存zadd
	//先合并table
	//再打散
	rest := make([]interface{}, len(ValueStringArr)+1)
	rest[0] = realTable
	for key, el := range ValueStringArr {
		rest[key+1] = el
	}
	_, err := conn.Do("Zadd", rest...)
	if err != nil {
		common.LogError("Redis ZAddStringArr zAdd has error", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// ZRevRangeStringArr 调用redis的zRevRange方法，
// request参数: table名,key,ValueStringArr数组必须有两个数分别代表查询几位到几位
// 返回：valueStringArr
func (obj *Redis) ZRevRangeStringArr(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//获取表名，key为时间，存储的信息
	table := request.GetTable()
	key := request.GetKey()
	number := request.ValueStringArr
	realTable := table + ":" + key
	if len(number) != 2 {
		common.LogError("Redis ZRevRangeStringArr Incoming err ValueStringArr length != 2")
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	reply := &pb.RedisMessage{}
	conn := obj.Pool.Get()
	defer conn.Close()

	byteSlices, err := redis.ByteSlices(conn.Do("zrevrange", realTable, number[0], number[1], "withScores"))
	if err != nil {
		common.LogError("Redis ZRevRangeStringArr zrevrange has error", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	ValueStringArr := make([]string, 0)
	for _, k := range byteSlices {
		ValueStringArr = append(ValueStringArr, string(k))
	}
	reply.ValueStringArr = ValueStringArr
	return reply, nil
}

// ZRemRangeByRankStringArr 调用redis的ZRemRangeByRank方法，
// request参数: table名，ValueStringArr数组必须有两个数分别代表删除几位到几位
// 返回：ValueBool
func (obj *Redis) ZRemRangeByRankStringArr(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//获取表名，key为时间，存储的信息
	table := request.GetTable()
	key := request.GetKey()
	number := request.ValueStringArr
	realTable := table + ":" + key
	if len(number) != 2 {
		common.LogError("Redis ZRemRangeByRankStringArr remove err ValueStringArr length != 2")
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	reply := &pb.RedisMessage{}
	conn := obj.Pool.Get()
	defer conn.Close()

	IsSuccess, err := redis.Bool(conn.Do("zremrangebyrank", realTable, number[0], number[1]))
	if err != nil {
		common.LogError("Redis ZRemRangeByRankStringArr zrevrange has error", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueBool = IsSuccess
	return reply, nil
}

// ZRevRankString 调用redis的zRevRange方法，
// request参数: table名,key,字段名（ValueString）
// 返回：ValueInt64
func (obj *Redis) ZRevRankString(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	//获取表名，key为时间，存储,用户uuid的信息
	table := request.GetTable()
	key := request.GetKey()
	realTable := table + ":" + key
	field := request.ValueString

	reply := &pb.RedisMessage{}
	conn := obj.Pool.Get()
	defer conn.Close()
	//获取字段索引
	rank, err := redis.Int64(conn.Do("ZREVRANK", realTable, field))
	if err != nil && err != redis.ErrNil {
		common.LogError("redis GetOneRankAndValue zrevrank has err:", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//当为空是
	if err == redis.ErrNil {
		reply.ValueInt64 = -1
		return reply, nil
	}
	reply.ValueInt64 = rank

	return reply, nil
}

//exists查询 键是否存在(通用)
//ValueBool 是 true 表示表存在
func (obj *Redis) Exists(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	// 获取传入的数据
	table := request.GetTable()
	key := request.GetKey()
	realTable := table + ":" + key
	// 连接池获取
	conn := obj.Pool.Get()
	defer conn.Close()
	reply := &pb.RedisMessage{}

	r, err := redis.Int64(conn.Do("exists", realTable))
	if err != nil {
		common.LogError("Redis Exists has err:", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	if r == 1 {
		reply.ValueBool = true
	}

	return reply, nil
}

//HGetAll 获取hash表里面的所有字段与值
//参数：table，key
//返回：ValueStringArr
func (obj *Redis) HGetAll(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	// 获取传入的数据
	table := request.GetTable()
	key := request.GetKey()
	realTable := table + ":" + key
	// 连接池获取
	conn := obj.Pool.Get()
	defer conn.Close()
	reply := &pb.RedisMessage{}

	r, err := redis.Strings(conn.Do("hgetall", realTable))
	if err != nil {
		common.LogError("Redis HGetAll has err:", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	if len(r)%2 != 0 {
		common.LogError("Redis HGetAll has err: Length is not a multiple of 2!")
	}
	reply.ValueStringArr = r
	return reply, nil
}

// IncrBy 对指定键增加指定数量并返回当前值
// 参数：table
// 返回：ValueStringArr
func (obj *Redis) IncrBy(request *pb.RedisMessage, extroInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	// 获取传入的数据
	table := request.GetTable()
	num := request.GetCount()
	// 连接池获取
	conn := obj.Pool.Get()
	defer conn.Close()
	reply := &pb.RedisMessage{}

	r, err := redis.Int64(conn.Do("incrby", table, num))
	if err != nil {
		common.LogError("Redis IncrBy has err:", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	// if len(r)%2 != 0 {
	// 	common.LogError("Redis HGetAll has err: Length is not a multiple of 2!")
	// }
	reply.ValueInt64 = r
	return reply, nil
}

// zCard 查询有序集合表的长度
// 参数：table
// 返回：ValueInt64
func (obj *Redis) ZCard(request *pb.RedisMessage, extraInfo *pb.MessageExtroInfo) (*pb.RedisMessage, *pb.ErrorMessage) {
	// 获取传入的数据
	table := request.GetTable()
	key := request.GetKey()
	realTable := table + ":" + key

	// 连接池获取
	conn := obj.Pool.Get()
	defer conn.Close()
	reply := &pb.RedisMessage{}

	r, err := redis.Int64(conn.Do("ZCard", realTable))

	if err != nil {
		common.LogError("Redis zCard has err:", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	reply.ValueInt64 = r
	return reply, nil
}

// HIncrBy 对 Redis.HIncrBy 的封装
// 参数
// table
// ValueString
func (obj *Redis) HIncrBy(
	request *pb.RedisMessage,
	extroInfo *pb.MessageExtroInfo,
) (*pb.RedisMessage, *pb.ErrorMessage) {
	// 获取传入的数据
	table := request.GetTable()
	key := request.GetKey()
	num := request.GetCount()
	// 连接池获取
	conn := obj.Pool.Get()
	defer conn.Close()
	reply := &pb.RedisMessage{}

	r, err := redis.Int64(conn.Do("hincrby", table, key, num))
	if err != nil {
		common.LogError("Redis HIncrBy has err:", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueInt64 = r
	return reply, nil
}

// HExists 查看哈希表 key 中，给定域 field 是否存在。
// 参数: key, field
// 返回: valueBool
func (obj *Redis) HExists(
	request *pb.RedisMessage,
	extroInfo *pb.MessageExtroInfo,
) (*pb.RedisMessage, *pb.ErrorMessage) {
	// 获取传入的数据
	key := request.GetTable()
	field := request.GetKey()
	// 连接池获取
	conn := obj.Pool.Get()
	defer conn.Close()
	reply := &pb.RedisMessage{}

	r, err := redis.Bool(conn.Do("hexists", key, field))
	if err != nil {
		common.LogError("Redis HExists has err:", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.ValueBool = r
	return reply, nil
}

func (obj *Redis) StartTrans() redis.Conn {
	conn := obj.Pool.Get()
	conn.Send("MULTI")
	return conn
}
func (obj *Redis) CommitTrans(conn redis.Conn) (reply interface{}, err error) {
	reply, err = conn.Do("EXEC")
	if err != nil {
		common.LogError("Redis CommitTrans redis has err", err)
		return reply, err
	}
	return reply, nil
}
func (obj *Redis) RollbackTrans(conn redis.Conn) error {
	_, err := conn.Do("DISCARD")
	if err != nil {
		common.LogError("Redis RollbackTrans redis has err", err)
		return err
	}
	return nil
}
