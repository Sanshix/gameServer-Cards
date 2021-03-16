package common

import (
	pb "gameServer-demo/src/grpc"

	"github.com/gogo/protobuf/proto"
	"github.com/gomodule/redigo/redis"
)

// Redis执行事务操作
func RedisExecTrans(transExecBody func(redisConn redis.Conn) *pb.ErrorMessage) (interface{}, *pb.ErrorMessage) {
	redisConn := Rediser.StartTrans()
	commitAll := false

	defer func(conn redis.Conn) {
		if !commitAll {
			err := Rediser.RollbackTrans(conn)
			if err != nil {
				LogError("redisExecTrans rollback redis trans error", err)
			}
		}
		conn.Close()
	}(redisConn)
	err := transExecBody(redisConn)
	if err != nil {
		LogError("transExecBody err", err)
		return nil, err
	}
	commitAll = true
	reply, redisErr := Rediser.CommitTrans(redisConn)
	if redisErr != nil {
		commitAll = false
		LogError("redisExecTrans 提交事务时出错", redisErr)
		return nil, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil

}

//redis 事物操作Sadd
func RedisTransSAdd(tableName, key string, values []string, conn redis.Conn) *pb.ErrorMessage {
	realTableName := tableName + ":" + key

	rest := make([]interface{}, len(values)+1)
	rest[0] = realTableName
	for key, el := range values {
		rest[key+1] = el
	}

	redisErr := conn.Send("SADD", rest...)
	if redisErr != nil {
		LogError("redisTransSAdd err", redisErr)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return nil
}

//redis 事物操作SRem
func RedisTransSRem(tableName, key string, values []string, conn redis.Conn) *pb.ErrorMessage {
	realTableName := tableName + ":" + key

	rest := make([]interface{}, len(values)+1)
	rest[0] = realTableName
	for key, el := range values {
		rest[key+1] = el
	}

	redisErr := conn.Send("SREM", rest...)
	if redisErr != nil {
		LogError("redisTransSRem err", redisErr)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return nil
}

//redis 事物操作HSet
func RedisTransHSet(tableName, key string, hashData proto.Message, conn redis.Conn) *pb.ErrorMessage {
	byteData, error := proto.Marshal(hashData)
	if error != nil {
		LogError("redisTransHSet proto.Marshal has err", error)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	redisErr := conn.Send("HSET", tableName, key, byteData)
	if redisErr != nil {
		LogError("redisTransHSet exec hset err", redisErr)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return nil
}

//redis 事物操作HSet
func RedisTransHGetString(tableName, key string, conn redis.Conn) (string, *pb.ErrorMessage) {
	ret, err := redis.String(conn.Do("HGET", tableName, key))
	if err == redis.ErrNil {
		return "", nil
	}
	return ret, nil
}

func RedisTransHGetByte(tableName, key string, conn redis.Conn) ([]byte, *pb.ErrorMessage) {

	var returnRet []byte
	ret, err := redis.Bytes(conn.Do("HGET", tableName, key))
	if err == redis.ErrNil {
		return returnRet, nil
	}
	if err != nil {
		LogError("Redis HGetByte has err", err)
		return returnRet, GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	returnRet = ret
	return returnRet, nil
}

//redis 事物操作HSet
func RedisTransHSetBytes(tableName, key string, bytes []byte, conn redis.Conn) *pb.ErrorMessage {
	redisErr := conn.Send("HSET", tableName, key, bytes)
	if redisErr != nil {
		LogError("redisTransHSet exec hset err", redisErr)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return nil
}

//redis 事物操作HSet
func RedisTransHSetString(tableName, key string, value string, conn redis.Conn) *pb.ErrorMessage {
	redisErr := conn.Send("HSET", tableName, key, value)
	if redisErr != nil {
		LogError("redisTransHSet exec hset err", redisErr)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return nil
}

//redis 事物操作HINCRBY
func RedisTransHINCRBY(tableName, key string, value int64, conn redis.Conn) *pb.ErrorMessage {
	err := conn.Send("HINCRBY", tableName, key, value)
	if err != nil {
		LogError("RedisTransHINCRBY exec HINCRBY err", err)
		return GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return nil
}
