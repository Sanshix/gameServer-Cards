package base

import (
	"database/sql"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/proto"
	uuid "github.com/satori/go.uuid"
)

func init() {
	common.AllComponentMap["Mysql"] = &Mysql{}
}

// Mysql 数据库组件
type Mysql struct {
	Base
	Db *sql.DB
}

// LoadComponent 加载组件
func (m *Mysql) LoadComponent(config *common.OneComponentConfig, componentName string) {
	m.Base.LoadComponent(config, componentName)
	dbTemp, err := sql.Open("mysql", (*m.Config)["connect_string"])
	if err != nil {
		panic(err)
	}
	// 链接超时时间
	dbTemp.SetConnMaxLifetime(600 * time.Second)
	// 空闲链接
	dbTemp.SetMaxIdleConns(0)
	//dbTemp.SetMaxOpenConns(30)
	m.Db = dbTemp
	// 初始化表结构
	err = m.checkTable()
	if err != nil {
		panic(err)
	}
	common.LogInfo("Mysql LoadComponent ok")
	return
}

// checkTable 执行检测表是否存在的sql
func (m *Mysql) checkTable() error {
	_, err := m.Db.Exec(common.MysqlCheckConfigGiftsTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlCheckConfigGiftsTable has err", err)
		return err
	}
	_, err = m.Db.Exec(common.MysqlCheckPlayerInfoTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlCheckPlayerInfoTable has err", err)
		return err
	}
	_, err = m.Db.Exec(common.MysqlCheckAccountInfoTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlCheckAccountInfoTable has err", err)
		return err
	}
	return nil
}

// NewAccount 新建账号
func (m *Mysql) NewAccount(request *pb.MysqlAccountInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	account := request.GetAccount()
	password := common.EncodePassword(request.GetPassword())
	mobile := request.GetMobile()

	shortID := request.GetShortId()
	reply := &pb.MysqlAccountInfo{}
	var checkUUID string
	var err error
	// 应为短信注册时account和mobile是一样的，所以这里即使是短信注册也可以只验证account
	err = m.Db.QueryRow("select uuid from "+common.MysqlAccountInfoTable+" where account = ?", account).Scan(&checkUUID)
	if err == nil {
		common.LogInfo("Mysql NewAccount account exist", account)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_AccountExist, "")
	}
	if err != nil && err != sql.ErrNoRows {
		common.LogError("Mysql NewAccount has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	userUUID := uuid.NewV4()

	nowTime := time.Now().Unix()
	if mobile != "" {
		_, err = m.Db.Exec("insert into "+common.MysqlAccountInfoTable+" (uuid,short_id,account,password,role_type,create_time,update_time,source_type,open_id,union_id,mobile) values(?,?,?,?,?,?,?,?,?,?,?)",
			userUUID.String(),
			shortID,
			account,
			password,
			request.RoleType,
			nowTime,
			nowTime,
			request.GetPlayerSourceType(),
			request.GetOpenId(),
			request.GetUnionId(),
			mobile)
	} else {
		_, err = m.Db.Exec("insert into "+common.MysqlAccountInfoTable+" (uuid,short_id,account,password,role_type,create_time,update_time,source_type,open_id,union_id) values(?,?,?,?,?,?,?,?,?,?)",
			userUUID.String(),
			shortID,
			account,
			password,
			request.RoleType,
			nowTime,
			nowTime,
			request.GetPlayerSourceType(),
			request.GetOpenId(),
			request.GetUnionId())
	}
	if err != nil {
		common.LogError("Mysql NewAccount insert has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	reply.Uuid = userUUID.String()
	reply.ShortId = shortID
	reply.Account = account
	return reply, nil
}

// ForceDeleteAccount 硬删除账号
func (m *Mysql) ForceDeleteAccount(request *pb.MysqlAccountInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	UUID := request.GetUuid()
	reply := &pb.MysqlAccountInfo{}
	_, err := m.Db.Exec("DELETE FROM "+common.MysqlAccountInfoTable+" WHERE uuid=?", UUID)
	if err != nil {
		common.LogError("Mysql ForceDeleteAccount has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// QueryAccountByShortID 通过shortid查询account
func (m *Mysql) QueryAccountByShortID(request *pb.MysqlAccountInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	reply := &pb.MysqlAccountInfo{}
	shortID := request.GetShortId()
	var mobile sql.NullString
	err := m.Db.QueryRow("select uuid,short_id,role_type,update_time,create_time,source_type,open_id,mobile,union_id from "+common.MysqlAccountInfoTable+" where short_id = ?",
		shortID).Scan(
		&reply.Uuid,
		&reply.ShortId,
		&reply.RoleType,
		&reply.UpdateTime,
		&reply.CreateTime,
		&reply.PlayerSourceType,
		&reply.OpenId,
		&mobile,
		&reply.UnionId)
	if mobile.Valid == true {
		reply.Mobile = mobile.String
	}
	if err == sql.ErrNoRows {
		common.LogInfo("Mysql QueryAccountByShortID account not found", shortID)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_DataNotFound, "用户名不存在或密码错误")
	}
	if err != nil {
		common.LogError("Mysql QueryAccountByShortID has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// QueryAccountByAccount 通过account查询account
func (m *Mysql) QueryAccountByAccount(request *pb.MysqlAccountInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	reply := &pb.MysqlAccountInfo{}
	account := request.GetAccount()
	var mobile sql.NullString
	err := m.Db.QueryRow("select uuid,short_id,role_type,update_time,source_type,open_id,mobile,union_id from "+common.MysqlAccountInfoTable+" where account = ?",
		account).Scan(
		&reply.Uuid,
		&reply.ShortId,
		&reply.RoleType,
		&reply.UpdateTime,
		&reply.PlayerSourceType,
		&reply.OpenId,
		&mobile,
		&reply.UnionId)
	if mobile.Valid == true {
		reply.Mobile = mobile.String
	}
	if err == sql.ErrNoRows {
		common.LogInfo("Mysql QueryAccountByAccount account not found", account)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_DataNotFound, "用户名不存在或密码错误")
	}
	if err != nil {
		common.LogError("Mysql QueryAccountByAccount has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// QueryAccountByUUID 通过uuid查询account
func (m *Mysql) QueryAccountByUUID(request *pb.MysqlAccountInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	reply := &pb.MysqlAccountInfo{}
	UUID := request.GetUuid()
	var mobile sql.NullString
	err := m.Db.QueryRow("select uuid,short_id,role_type,update_time,create_time,source_type,open_id,mobile,union_id from "+common.MysqlAccountInfoTable+" where uuid = ?",
		UUID).Scan(
		&reply.Uuid,
		&reply.ShortId,
		&reply.RoleType,
		&reply.UpdateTime,
		&reply.CreateTime,
		&reply.PlayerSourceType,
		&reply.OpenId,
		&mobile,
		&reply.UnionId)
	if mobile.Valid == true {
		reply.Mobile = mobile.String
	}
	if err == sql.ErrNoRows {
		common.LogInfo("Mysql QueryAccountByUUID account not found", UUID)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_DataNotFound, "用户名不存在或密码错误")
	}
	if err != nil {
		common.LogError("Mysql QueryAccountByUUID has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// QueryAccountByMobile 通过mobile查询account
func (m *Mysql) QueryAccountByMobile(request *pb.MysqlAccountInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	reply := &pb.MysqlAccountInfo{}
	wantMobile := request.GetMobile()
	var mobile sql.NullString
	err := m.Db.QueryRow("select uuid,short_id,role_type,update_time,create_time,source_type,open_id,mobile,union_id from "+common.MysqlAccountInfoTable+" where mobile = ?",
		wantMobile).Scan(
		&reply.Uuid,
		&reply.ShortId,
		&reply.RoleType,
		&reply.UpdateTime,
		&reply.CreateTime,
		&reply.PlayerSourceType,
		&reply.OpenId,
		&mobile,
		&reply.UnionId)
	if mobile.Valid == true {
		reply.Mobile = mobile.String
	}
	if err == sql.ErrNoRows {
		common.LogInfo("Mysql QueryAccountByUUID account not found", wantMobile)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_DataNotFound, "用户名不存在或密码错误")
	}
	if err != nil {
		common.LogError("Mysql QueryAccountByUUID has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// QueryStressTestAccount 查询压测账号
func (m *Mysql) QueryStressTestAccount(request *pb.QueryAccountByRoleRequest, extroInfo *pb.MessageExtroInfo) (*pb.QueryAccountByRoleReply, *pb.ErrorMessage) {
	reply := &pb.QueryAccountByRoleReply{}
	rows, err := m.Db.Query("select uuid,short_id,account from " + common.MysqlAccountInfoTable + " where account like 'stressTestRobot-%'")
	defer rows.Close()
	if err == sql.ErrNoRows {
		return reply, nil
	}
	if err != nil {
		common.LogError("Mysql QueryStressTestAccount has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	allInfo := []*pb.MysqlAccountInfo{}
	for rows.Next() {
		oneInfo := &pb.MysqlAccountInfo{}
		err = rows.Scan(
			&oneInfo.Uuid,
			&oneInfo.ShortId,
			&oneInfo.Account)
		if err != nil {
			common.LogError("Mysql QueryStressTestAccount rows Scan has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		allInfo = append(allInfo, oneInfo)
	}
	reply.AccountInfos = allInfo
	return reply, nil
}

// VerifyAccount 验证账号的用户名和密码
func (m *Mysql) VerifyAccount(request *pb.MysqlAccountInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	account := request.GetAccount()
	password := common.EncodePassword(request.GetPassword())
	wantMobile := request.GetMobile()
	reply := &pb.MysqlAccountInfo{}
	sqlStr := "select uuid,short_id,role_type,update_time,create_time,source_type,open_id,mobile,union_id from " + common.MysqlAccountInfoTable + " where account = ? and password = ?"
	// 手机登陆
	if wantMobile != "" {
		sqlStr = "select uuid,short_id,role_type,update_time,create_time,source_type,open_id,mobile,union_id from " + common.MysqlAccountInfoTable + " where mobile = ? and password = ?"
		account = wantMobile
	}
	var mobile sql.NullString
	err := m.Db.QueryRow(sqlStr,
		account, password).Scan(
		&reply.Uuid,
		&reply.ShortId,
		&reply.RoleType,
		&reply.UpdateTime,
		&reply.CreateTime,
		&reply.PlayerSourceType,
		&reply.OpenId,
		&mobile,
		&reply.UnionId)
	if mobile.Valid == true {
		reply.Mobile = mobile.String
	}
	if err == sql.ErrNoRows {
		common.LogInfo("Mysql VerifyAccount account not found", account, password)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_DataNotFound, "用户名不存在或密码错误")
	}
	if err != nil {
		common.LogError("Mysql VerifyAccount has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// QueryAccountByRole 通过角色类型查找玩家
func (m *Mysql) QueryAccountByRole(request *pb.QueryAccountByRoleRequest, extroInfo *pb.MessageExtroInfo) (*pb.QueryAccountByRoleReply, *pb.ErrorMessage) {
	roleType := request.GetRoleType()
	reply := &pb.QueryAccountByRoleReply{}
	rows, err := m.Db.Query("select uuid,short_id,account from "+common.MysqlAccountInfoTable+" where role_type = ?", roleType)
	defer rows.Close()
	if err == sql.ErrNoRows {
		return reply, nil
	}
	if err != nil {
		common.LogError("Mysql QueryAccountByRole has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	allInfo := []*pb.MysqlAccountInfo{}
	for rows.Next() {
		oneInfo := &pb.MysqlAccountInfo{}
		err = rows.Scan(
			&oneInfo.Uuid,
			&oneInfo.ShortId,
			&oneInfo.Account)
		if err != nil {
			common.LogError("Mysql rows Scan has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		allInfo = append(allInfo, oneInfo)
	}
	reply.AccountInfos = allInfo
	return reply, nil
}

// QueryThirdPartyAccount 通过角色类型查找玩家
func (m *Mysql) QueryThirdPartyAccount(request *pb.QueryThirdPartyAccountRequest, extroInfo *pb.MessageExtroInfo) (*pb.QueryThirdPartyAccountReply, *pb.ErrorMessage) {
	reply := &pb.QueryThirdPartyAccountReply{}

	queryArgs := make([]interface{}, 0)

	querySql := "select uuid,short_id,account,role_type,update_time,create_time,source_type,open_id,mobile,union_id from " + common.MysqlAccountInfoTable + " where source_type = ? and open_id=?"

	queryArgs = append(queryArgs, int32(request.GetPlayerSourceType()))
	queryArgs = append(queryArgs, request.GetOpenId())
	if request.GetUnionId() != "" {
		querySql += " and union_id=?"
		queryArgs = append(queryArgs, request.GetUnionId())
	}

	rows, err := m.Db.Query(querySql, queryArgs...)
	defer rows.Close()
	if err == sql.ErrNoRows {
		return reply, nil
	}
	if err != nil {
		common.LogError("Mysql QueryThirdPartyAccount has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	var allInfo []*pb.MysqlAccountInfo
	for rows.Next() {
		oneInfo := &pb.MysqlAccountInfo{}
		var mobile sql.NullString
		err = rows.Scan(
			&oneInfo.Uuid,
			&oneInfo.ShortId,
			&oneInfo.Account,
			&oneInfo.RoleType,
			&oneInfo.UpdateTime,
			&oneInfo.CreateTime,
			&oneInfo.PlayerSourceType,
			&oneInfo.OpenId,
			&mobile,
			&oneInfo.UnionId)
		if mobile.Valid == true {
			oneInfo.Mobile = mobile.String
		}
		if err != nil {
			common.LogError("Mysql rows Scan has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		allInfo = append(allInfo, oneInfo)
	}
	reply.AccountInfos = allInfo
	return reply, nil
}

// QueryKV kv类型表的查询操作
func (m *Mysql) QueryKV(request *pb.MysqlKVMessage, extroInfo *pb.MessageExtroInfo) (*pb.MysqlKVMessage, *pb.ErrorMessage) {
	uuid := request.GetUuid()
	shortID := request.GetShortId()
	tableName := request.GetTableName()
	reply := &pb.MysqlKVMessage{}
	var err error
	if uuid != "" {
		err = m.Db.QueryRow("select info from "+tableName+" where uuid = ?", uuid).Scan(&reply.Info)
	} else if shortID != "" {
		err = m.Db.QueryRow("select info from "+tableName+" where short_id = ?", shortID).Scan(&reply.Info)
	}
	if err == sql.ErrNoRows {
		common.LogInfo("Mysql QueryKV data not found", uuid, shortID, tableName)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_DataNotFound, "")
	}
	if err != nil {
		common.LogError("Mysql QueryKV has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// InsertKV kv类型表的插入操作
func (m *Mysql) InsertKV(request *pb.MysqlKVMessage, extroInfo *pb.MessageExtroInfo) (*pb.MysqlKVMessage, *pb.ErrorMessage) {
	uuid := request.GetUuid()
	shortID := request.GetShortId()
	tableName := request.GetTableName()
	info := request.GetInfo()
	nowTime := time.Now().Unix()
	reply := &pb.MysqlKVMessage{}
	_, err := m.Db.Exec("insert into "+tableName+" (uuid,short_id,info,update_time) values(?,?,?,?)", uuid, shortID, info, nowTime)
	if err != nil {
		common.LogError("Mysql InsertKV insert has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// UpdateKV kv类型表的更新操作
func (m *Mysql) UpdateKV(request *pb.MysqlKVMessage, extroInfo *pb.MessageExtroInfo) (*pb.MysqlKVMessage, *pb.ErrorMessage) {
	uuid := request.GetUuid()
	tableName := request.GetTableName()
	info := request.GetInfo()
	nowTime := time.Now().Unix()
	reply := &pb.MysqlKVMessage{}
	_, err := m.Db.Exec("update "+tableName+" set info=?,update_time=? where uuid=?", info, nowTime, uuid)
	if err != nil {
		common.LogError("Mysql UpdateKV insert has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// DeleteKV kv类型表的删除操作
func (m *Mysql) DeleteKV(request *pb.MysqlKVMessage, extroInfo *pb.MessageExtroInfo) (*pb.MysqlKVMessage, *pb.ErrorMessage) {
	uuid := request.GetUuid()
	tableName := request.GetTableName()
	reply := &pb.MysqlKVMessage{}
	_, err := m.Db.Exec("delete "+tableName+" where uuid=?", uuid)
	if err != nil {
		common.LogError("Mysql DeleteKV  has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// GetAccountPageList 获取账户分页列表
func (m *Mysql) GetAccountPageList(request *pb.GetAccountPageListRequest, extraInfo *pb.MessageExtroInfo) (*pb.GetAccountPageListReply, *pb.ErrorMessage) {
	reply := &pb.GetAccountPageListReply{}
	reply.PageIndex = request.PageIndex
	reply.PageSize = request.PageSize
	queryCountSQL := "select count(1) cnt from " + common.MysqlAccountInfoTable + " where role_type=? and account not like '%_deleted'"
	//查询列表
	queryListSQL := `select a.auto_id,a.uuid,a.short_id,a.account,a.role_type,a.source_type,a.open_id,a.union_id,a.create_time,a.update_time,b.info playerInfo from ` + common.MysqlAccountInfoTable + ` a left join ` + common.MysqlPlayerInfoTable + ` b  on a.short_id=b.short_id where a.role_type=? and account not like '%_deleted'`
	args := make([]interface{}, 1)
	args[0] = request.RoleType
	if request.AccountId != "" {
		queryCountSQL += " and short_id=?"
		queryListSQL += " and a.short_id=?"
		args = append(args, request.AccountId)
	}
	if request.SourceType != pb.PlayerSourceType_PlayerSourceType_None {
		queryCountSQL += " and source_type=?"
		queryListSQL += " and a.source_type=?"
		args = append(args, request.SourceType)
	}
	if request.Account != "" {
		queryCountSQL += " and account=?"
		queryListSQL += " and a.account=?"
		args = append(args, request.Account)
	}

	queryListSQL += ` order by a.update_time desc limit ` + strconv.FormatInt(int64(request.PageSize*(request.PageIndex-1)), 10) + "," + strconv.FormatInt(int64(request.PageSize), 10)

	common.LogDebug("sql:", queryCountSQL, queryListSQL)
	var rowCount int32
	err := m.Db.QueryRow(
		queryCountSQL,
		args...).Scan(&rowCount)
	if err != nil {
		common.LogError("GetAccountPageList error:", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	if rowCount == 0 {
		return reply, nil
	}

	reply.RecordCount = rowCount

	rows, err := m.Db.Query(queryListSQL, args...)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("GetAccountPageList has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		return reply, nil
	}
	reply.Data = make([]*pb.AccountDetailReply, 0)

	for {
		if !rows.Next() {
			break
		}
		out := &pb.AccountDetailReply{}
		playerInfoByte := make([]byte, 0)
		err = rows.Scan(
			&out.AutoId,
			&out.Uuid,
			&out.ShortId,
			&out.Account,
			&out.RoleType,
			&out.SourceType,
			&out.OpenId,
			&out.UnionId,
			&out.CreateTime,
			&out.UpdateTime,
			&playerInfoByte)
		if err != nil {
			common.LogError("GetAccountPageList has err", err)
			continue
		}
		playerInfo := &pb.PlayerInfo{}
		err = proto.Unmarshal(playerInfoByte, playerInfo)
		if err != nil {
			common.LogError("GetAccountPageList parse playerInfo has err", err)
			continue
		}
		out.Status = playerInfo.Status
		if playerInfo.Status == pb.UserStatus_invalid {
			out.Status = pb.UserStatus_normal
		}
		out.Name = playerInfo.Name
		out.Balance = playerInfo.Balance
		out.IsRobot = playerInfo.IsRobot
		out.GameType = playerInfo.GameType
		out.GameScene = playerInfo.GameScene
		out.GameServerIndex = playerInfo.GameServerIndex
		out.RoomId = playerInfo.RoomId
		out.LastRoomId = playerInfo.LastRoomId
		out.Auths = playerInfo.Auths
		out.LastLoginTime = playerInfo.LastLoginTime

		//out.PlayerInfo = playerInfo
		reply.Data = append(reply.Data, out)
	}
	return reply, nil
}

// UpdateAccountMobile 更新用户账户的手机号
func (m *Mysql) UpdateAccountMobile(request *pb.MysqlAccountInfo, extraInfo *pb.MessageExtroInfo) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	reply := &pb.MysqlAccountInfo{}

	if request.GetMobile() == "" {
		common.LogError("Mysql UpdateAccountMobile request.GetMobile is nil")
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	nowTime := time.Now().Unix()

	password := common.EncodePassword(request.GetPassword())

	_, err := m.Db.Exec("update "+common.MysqlAccountInfoTable+" set mobile=?,update_time=?,password=? where uuid=?",
		request.GetMobile(),
		&nowTime,
		&password,
		request.GetUuid())
	if err != nil {
		common.LogError("Mysql UpdateAccountMobile update has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// UpdateAccountInfo 更新用户账户信息
func (m *Mysql) UpdateAccountInfo(request *pb.MysqlAccountInfo, extraInfo *pb.MessageExtroInfo) (*pb.MysqlAccountInfo, *pb.ErrorMessage) {
	reply := &pb.MysqlAccountInfo{}
	err := m.Db.QueryRow("select uuid,short_id,password,role_type,update_time from "+common.MysqlAccountInfoTable+" where uuid = ? ",
		request.Uuid).Scan(
		&reply.Uuid,
		&reply.ShortId,
		&reply.Password,
		&reply.RoleType,
		&reply.UpdateTime)
	if err == sql.ErrNoRows {
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_DataNotFound, "")
	}
	if err != nil {
		common.LogError("UpdateAccountInfo Get AccountInfo err", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	if request.Password != "" {
		reply.Password = common.EncodePassword(request.Password)
	}
	reply.UpdateTime = time.Now().Unix()

	_, err = m.Db.Exec("update "+common.MysqlAccountInfoTable+" set password=?,update_time=? where uuid=?",
		&reply.Password,
		&reply.UpdateTime,
		&reply.Uuid)

	if err != nil {
		common.LogError("Mysql InsertRBT insert has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// SetAccountStatus 设置用户状态
func (m *Mysql) SetAccountStatus(request *pb.SetAccountStatusRequest, extraInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	strSQL := `update ` + common.MysqlAccountInfoTable + " set status=? where uuid=?"

	_, err := m.Db.Exec(strSQL, request.Status, request.Uuid)
	if err != nil {
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return &pb.EmptyMessage{}, nil
}

// DeleteAccount 删除账户
func (m *Mysql) DeleteAccount(request *pb.DeleteAccountRequest, extraInfo *pb.MessageExtroInfo) (*pb.DeleteAccountReply, *pb.ErrorMessage) {
	getAccountInfoSQL := "select short_id,account from " + common.MysqlAccountInfoTable + " where uuid=?"
	accountInfo := &pb.MysqlAccountInfo{}
	reply := &pb.DeleteAccountReply{Uuid: request.Uuid}
	error := m.Db.QueryRow(getAccountInfoSQL, request.Uuid).Scan(&accountInfo.ShortId, &accountInfo.Account)
	if error != nil {
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_AccountExist, "")
	}

	strSQL := `update ` + common.MysqlAccountInfoTable + " set status=?,account=? where uuid=?"

	_, err := m.Db.Exec(strSQL, pb.UserStatus_deleted, accountInfo.Account+"_deleted", request.Uuid)
	if err != nil {
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// =============================================================================
// TODO: 俱乐部数据库操作
// =============================================================================

// InsertClubData 插入俱乐部数据
func (m *Mysql) InsertClubData(request *pb.ClubInfo, extraInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	reply := &pb.EmptyMessage{}
	strSql := `insert into ` + common.MysqlClubInfoTable +
		` (uuid,short_id,name,master_uuid,status,create_time,update_time,info)` +
		` values(?,?,?,?,?,?,?,?)`
	queryArgs := make([]interface{}, 0)
	queryArgs = append(queryArgs, request.UUID)
	queryArgs = append(queryArgs, request.InviteCode)
	queryArgs = append(queryArgs, request.Name)
	queryArgs = append(queryArgs, request.MasterUUID)
	queryArgs = append(queryArgs, request.Status)
	queryArgs = append(queryArgs, request.CreateTime)
	queryArgs = append(queryArgs, time.Now().Unix())
	clubInfoByte, err := proto.Marshal(request)
	if err != nil {
		common.LogError("InsertClubData marshal detail has error", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	queryArgs = append(queryArgs, clubInfoByte)
	_, err = m.Db.Exec(strSql, queryArgs...)
	if err != nil {
		common.LogError("InsertClubData  has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// UpdateClubInfo更新俱乐部信息
func (m *Mysql) UpdateClubInfo(clubInfo *pb.ClubInfo, extraInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	reply := &pb.EmptyMessage{}
	strSql := `update ` + common.MysqlClubInfoTable +
		` set name=?,short_id=?,master_uuid=?,status=?,update_time=?,info=?` +
		` where uuid=?`
	queryArgs := make([]interface{}, 0)
	queryArgs = append(queryArgs, clubInfo.Name)
	queryArgs = append(queryArgs, clubInfo.InviteCode)
	queryArgs = append(queryArgs, clubInfo.MasterUUID)
	queryArgs = append(queryArgs, clubInfo.Status)
	queryArgs = append(queryArgs, time.Now().Unix())
	clubInfoByte, err := proto.Marshal(clubInfo)
	if err != nil {
		common.LogError("UpdateClubInfo marshal detail has error", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	queryArgs = append(queryArgs, clubInfoByte)
	queryArgs = append(queryArgs, clubInfo.UUID)
	_, err = m.Db.Exec(strSql, queryArgs...)
	if err != nil {
		common.LogError("UpdateClubInfo  has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// GetClubInfo 根据uuid获取俱乐部信息
func (m *Mysql) GetClubInfo(request *pb.QueryClubInfoRequest, extraInfo *pb.MessageExtroInfo) (*pb.ClubInfo, *pb.ErrorMessage) {
	strSql := "select info from " + common.MysqlClubInfoTable +
		" where 1=1 "
	//" where status<> " + strconv.Itoa(int(pb.ClubStatus_MasterDismiss))
	queryArgs := make([]interface{}, 0)
	if request.ClubUUID != "" {
		strSql += " and uuid=?"
		queryArgs = append(queryArgs, request.ClubUUID)
	}
	if request.InviteCode != "" {
		strSql += " and short_id=?"
		queryArgs = append(queryArgs, request.InviteCode)
	}
	if len(queryArgs) == 0 {
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_InvalidParameters, "")
	}

	detailInfoByte := make([]byte, 0)
	err := m.Db.QueryRow(
		strSql,
		queryArgs...).Scan(&detailInfoByte)
	detailData := &pb.ClubInfo{}
	err = proto.Unmarshal(detailInfoByte, detailData)
	if err != nil {
		common.LogError("GetLeagueData parse LeagueDetail has err", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return detailData, nil
}

// GetClubPageList 获取俱乐部分页列表
func (m *Mysql) GetClubPageList(request *pb.GetClubPageListRequest, extraInfo *pb.MessageExtroInfo) (*pb.GetClubPageListReply, *pb.ErrorMessage) {
	reply := &pb.GetClubPageListReply{}
	reply.PageIndex = request.PageIndex
	reply.PageSize = request.PageSize
	queryCountSQL := "select count(1) cnt from " + common.MysqlClubInfoTable + " a inner join " + common.MysqlAccountInfoTable + " b on a.master_uuid=b.uuid where 1=1"
	//查询列表
	queryListSQL := `select b.short_id,b.account,a.info clubInfo from ` + common.MysqlClubInfoTable + ` a left join ` + common.MysqlAccountInfoTable + ` b  on b.uuid=a.master_uuid where 1=1`
	args := make([]interface{}, 0)
	if request.ClubUUID != "" {
		queryCountSQL += " and a.uuid=?"
		queryListSQL += " and a.uuid=?"
		args = append(args, request.ClubUUID)
	}
	if request.InviteCode != "" {
		queryCountSQL += " and a.short_id=?"
		queryListSQL += " and a.short_id=?"
		args = append(args, request.InviteCode)
	}
	if request.Status != pb.ClubStatus_ClubNone {
		queryCountSQL += " and a.status=?"
		queryListSQL += " and a.status=?"
		args = append(args, request.Status)
	}
	if request.ClubName != "" {
		queryCountSQL += " and a.name=?"
		queryListSQL += " and a.name=?"
		args = append(args, request.ClubName)
	}
	if request.MasterShortId != "" {
		queryCountSQL += " and b.short_id=?"
		queryListSQL += " and b.short_id=?"
		args = append(args, request.MasterShortId)
	}
	if request.MasterAccount != "" {
		queryCountSQL += " and b.account=?"
		queryListSQL += " and b.account=?"
		args = append(args, request.MasterAccount)
	}

	queryListSQL += ` order by b.update_time desc limit ` + strconv.FormatInt(int64(request.PageSize*(request.PageIndex-1)), 10) + "," + strconv.FormatInt(int64(request.PageSize), 10)

	common.LogDebug("sql:", queryCountSQL, queryListSQL)
	var rowCount int32
	err := m.Db.QueryRow(
		queryCountSQL,
		args...).Scan(&rowCount)
	if err != nil {
		common.LogError("GetClubPageList error:", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	if rowCount == 0 {
		return reply, nil
	}

	reply.RecordCount = rowCount

	rows, err := m.Db.Query(queryListSQL, args...)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("GetClubPageList has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		return reply, nil
	}
	reply.Data = make([]*pb.GetClubPageDataDetail, 0)

	for {
		if !rows.Next() {
			break
		}
		out := &pb.GetClubPageDataDetail{}
		clubInfoByte := make([]byte, 0)
		err = rows.Scan(
			&out.MasterShortId,
			&out.MasterAccount,
			&clubInfoByte)
		if err != nil {
			common.LogError("GetClubPageList has err", err)
			continue
		}
		clubInfo := &pb.ClubInfo{}
		err = proto.Unmarshal(clubInfoByte, clubInfo)
		if err != nil {
			common.LogError("GetClubPageList parse playerInfo has err", err)
			continue
		}
		out.Status = clubInfo.Status
		out.UUID = clubInfo.UUID
		out.InviteCode = clubInfo.InviteCode
		out.Name = clubInfo.Name
		out.MasterUUID = clubInfo.MasterUUID
		out.CreateTime = clubInfo.CreateTime
		out.CloseTime = clubInfo.CloseTime
		out.MemberList = clubInfo.MemberList
		out.RoomList = clubInfo.RoomList

		reply.Data = append(reply.Data, out)
	}
	return reply, nil
}

// =============================================================================
// TODO: 大联盟数据库操作
// =============================================================================

// InsertCreateLeagueData 插入创建联盟数据
func (m *Mysql) InsertCreateLeagueData(request *pb.InsertLeague2MysqlRequest, extraInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	reply := &pb.EmptyMessage{}
	strSql := `insert into ` + common.MysqlLeagueInfoTable +
		` (uuid,invite_code,name,master_uuid,status,create_time,details,status_change_time,last_update_time)` +
		` values(?,?,?,?,?,?,?,?,?)`
	queryArgs := make([]interface{}, 0)
	queryArgs = append(queryArgs, request.Data.UUID)
	queryArgs = append(queryArgs, request.Data.InviteCode)
	queryArgs = append(queryArgs, request.Data.Name)
	queryArgs = append(queryArgs, request.Data.MasterUUID)
	queryArgs = append(queryArgs, request.Data.Status)
	queryArgs = append(queryArgs, request.Data.CreateTime)
	leagueInfoByte, err := proto.Marshal(request.Data.Details)
	if err != nil {
		common.LogError("InsertCreateLeagueData marshal detail has error", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_LeagueErrInsertLeagueInfoFail, "")
	}
	queryArgs = append(queryArgs, leagueInfoByte)
	queryArgs = append(queryArgs, request.Data.StatusChangeTime)
	queryArgs = append(queryArgs, request.Data.LastUpdateTime)
	_, err = m.Db.Exec(strSql, queryArgs...)
	if err != nil {
		common.LogError("InsertCreateLeagueData  has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// UpdateLeagueData 插入创建联盟数据
func (m *Mysql) UpdateLeagueData(request *pb.InsertLeague2MysqlRequest, extraInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	reply := &pb.EmptyMessage{}
	strSql := `update ` + common.MysqlLeagueInfoTable +
		` set status=?,details=?,status_change_time=?,last_update_time=? where uuid=?`
	leagueInfoByte, err := proto.Marshal(request.Data.Details)
	if err != nil {
		common.LogError("UpdateLeagueData marshal detail has error", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_LeagueErrInsertLeagueInfoFail, "")
	}
	queryArgs := make([]interface{}, 0)
	queryArgs = append(queryArgs, request.Data.Status)
	queryArgs = append(queryArgs, leagueInfoByte)
	queryArgs = append(queryArgs, request.Data.StatusChangeTime)
	queryArgs = append(queryArgs, request.Data.LastUpdateTime)
	queryArgs = append(queryArgs, request.Data.UUID)

	_, err = m.Db.Exec(strSql, queryArgs...)
	if err != nil {
		common.LogError("UpdateLeagueData  has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil
}

// GetLeagueData 获取联盟数据
func (m *Mysql) GetLeagueData(request *pb.GetLeagueFromMysqlRequest, extraInfo *pb.MessageExtroInfo) (*pb.LeagueInfo, *pb.ErrorMessage) {
	reply := &pb.LeagueInfo{}
	strSql := `select uuid,invite_code,name,master_uuid,status,create_time,details,status_change_time,last_update_time from ` +
		common.MysqlLeagueInfoTable +
		` where uuid=?`

	detailInfoByte := make([]byte, 0)
	err := m.Db.QueryRow(
		strSql,
		request.Uuid).Scan(
		&reply.UUID,
		&reply.InviteCode,
		&reply.Name,
		&reply.MasterUUID,
		&reply.Status,
		&reply.CreateTime,
		&detailInfoByte,
		&reply.StatusChangeTime,
		&reply.LastUpdateTime,
	)
	detailData := &pb.LeagueDetail{}
	err = proto.Unmarshal(detailInfoByte, detailData)
	if err != nil {
		common.LogError("GetLeagueData parse LeagueDetail has err", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply.Details = detailData
	return reply, nil
}

// QueryOrderList 分页查询订单列表
func (m *Mysql) QueryOrderList(request *pb.GetOrderPageListRequest, extroInfo *pb.MessageExtroInfo) (*pb.GetOrderPageListReply, *pb.ErrorMessage) {
	reply := &pb.GetOrderPageListReply{}
	reply.PageIndex = request.PageIndex
	reply.PageSize = request.PageSize
	queryCountSQL := "select count(1) cnt from " + common.MysqlOrderInfoTable + " o  LEFT JOIN   game_account_info a on a.uuid=o.player_uuid where 1=1 "
	//查询列表

	queryListSQL := `SELECT a.short_id,o.auto_id,o.uuid,o.player_uuid,o.txn_id,o.order_amount,o.order_state,o.product_type,o.product_num,o.product_uuid,o.channel,o.remark,o.update_time,o.create_time,o.currency
 FROM ` + common.MysqlOrderInfoTable + ` o  LEFT JOIN   game_account_info a on
a.uuid=o.player_uuid where  1=1 `
	args := make([]interface{}, 0)
	if request.OrderState != pb.OrderStatus_OrderStatus_None {
		queryCountSQL += " and o.order_state=?"
		queryListSQL += " and o.order_state=?"
		args = append(args, request.OrderState)
	}
	if request.ProductType != pb.ProductType_ProductType_None {
		queryCountSQL += " and o.product_type=?"
		queryListSQL += " and o.product_type=?"
		args = append(args, request.ProductType)
	}
	if request.ShortId != "" {
		queryCountSQL += "and a.short_id=?"
		queryListSQL += " and a.short_id=?"
		args = append(args, request.ShortId)
	}

	queryListSQL += ` order by o.create_time desc limit ` + strconv.FormatInt(int64(request.PageSize*(request.PageIndex-1)), 10) + "," + strconv.FormatInt(int64(request.PageSize), 10)

	common.LogDebug("sql:", queryCountSQL, queryListSQL)
	var rowCount int32
	err := m.Db.QueryRow(
		queryCountSQL,
		args...).Scan(&rowCount)
	if err != nil {
		common.LogError("GetOrderPageList error:", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	if rowCount == 0 {
		return reply, nil
	}
	reply.RecordCount = rowCount
	rows, err := m.Db.Query(queryListSQL, args...)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("GetAccountPageList has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		return reply, nil
	}
	reply.Data = make([]*pb.OrderDetailReply, 0)
	for {
		if !rows.Next() {
			break
		}
		out := &pb.OrderDetailReply{}
		err = rows.Scan(
			&out.ShortId,
			&out.AutoId,
			&out.Uuid,
			&out.PlayerUuid,
			&out.TxnId,
			&out.OrderAmount,
			&out.OrderState,
			&out.ProductType,
			&out.ProductNum,
			&out.ProductUuid,
			&out.Channel,
			&out.Remark,
			&out.UpdateTime,
			&out.CreateTime,
			&out.Currency,
		)
		if err != nil {
			common.LogError("GetOrderPageList has err", err)
			continue
		}
		reply.Data = append(reply.Data, out)
	}
	return reply, nil

}

// NewOrder 新建订单
func (m *Mysql) NewOrder(request *pb.MysqlOrderInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlOrderInfo, *pb.ErrorMessage) {

	nowTime := time.Now().Unix()
	_, err := m.Db.Exec("insert into "+common.MysqlOrderInfoTable+
		" (uuid,player_uuid,order_amount,order_state,product_uuid,product_type,product_num,channel,currency,create_time,update_time,remark,txn_id) values(?,?,?,?,?,?,?,?,?,?,?,?,?)",
		request.Uuid,
		request.PlayerUuid,
		request.OrderAmount,
		request.OrderState,
		request.ProductUuid,
		request.ProductType,
		request.ProductNum,
		request.Channel,
		request.Currency,
		nowTime,
		nowTime,
		request.Remark,
		request.TxnId,
	)
	if err != nil {
		common.LogError("Mysql NewOrder insert has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	return request, nil
}

// UpdateOrder 更新订单
func (m *Mysql) UpdateOrder(request *pb.MysqlOrderInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlOrderInfo, *pb.ErrorMessage) {

	request.UpdateTime = time.Now().Unix()
	_, err := m.Db.Exec("update "+common.MysqlOrderInfoTable+" set txn_id=?,order_state=?,remark=?,update_time=? where uuid=?",
		&request.TxnId,
		&request.OrderState,
		&request.Remark,
		&request.UpdateTime,
		&request.Uuid)

	if err != nil {
		common.LogError("Mysql UpdateOrder  has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return request, nil
}

// GetOrderByUUID 查询一条订单
func (m *Mysql) GetOrderByUUID(request *pb.MysqlOrderInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlOrderInfo, *pb.ErrorMessage) {
	reply := &pb.MysqlOrderInfo{}
	err := m.Db.QueryRow("select auto_id,uuid,player_uuid,txn_id,order_amount,order_state,product_uuid,product_type,product_num,channel,create_time,update_time,remark,currency from "+common.MysqlOrderInfoTable+" where uuid = ? ",
		request.Uuid).Scan(
		&reply.AutoId,
		&reply.Uuid,
		&reply.PlayerUuid,
		&reply.TxnId,
		&reply.OrderAmount,
		&reply.OrderState,
		&reply.ProductUuid,
		&reply.ProductType,
		&reply.ProductNum,
		&reply.Channel,
		&reply.CreateTime,
		&reply.UpdateTime,
		&reply.Remark,
		&reply.Currency,
	)

	if err == sql.ErrNoRows {
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_DataNotFound, "")
	}
	if err != nil {
		common.LogError("Mysql  GetOrderByUUID  err", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	return reply, nil
}

// GetOrderGoldCountByUUID 根据订单查询用户累计充值记录
func (m *Mysql) GetOrderGoldCountByUUID(playerInfo *pb.PlayerInfo, extroInfo *pb.MessageExtroInfo) (*pb.MysqlOrderInfo, *pb.ErrorMessage) {

	var SumOrderAmount sql.NullInt64

	err := m.Db.QueryRow("SELECT SUM(order_amount) as sum_order_amount FROM "+common.MysqlOrderInfoTable+" WHERE order_state = 2 and player_uuid = ?",
		playerInfo.Uuid).Scan(
		&SumOrderAmount,
	)
	if err != nil {
		common.LogError("Mysql  GetOrderGoldCountByUUID  err", err)
		return nil, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	reply := &pb.MysqlOrderInfo{
		ProductNum: SumOrderAmount.Int64,
	}
	return reply, nil
}

// GetNoticeList 查询公告信息
func (m *Mysql) GetNoticeList(request *pb.GetNoticeRequest, extroInfo *pb.MessageExtroInfo) (*pb.GetNoticeReply, *pb.ErrorMessage) {
	reply := &pb.GetNoticeReply{}
	strSql := `select id,title,content,create_time,update_time,hide,order_sort from ` +
		common.MysqlConfigNotice + ` order by order_sort DESC`

	rows, err := m.Db.Query(strSql)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("GetNoticeList has err", err)
			return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		return reply, nil
	}
	for {
		if !rows.Next() {
			break
		}
		out := &pb.Notice{}
		err = rows.Scan(
			&out.Id,
			&out.Title,
			&out.Content,
			&out.CreateTime,
			&out.UpdateTime,
			&out.Hide,
			&out.OrderSort,
		)
		if err != nil {
			common.LogError("GetNoticeList has err", err)
			continue
		}
		reply.Data = append(reply.Data, out)
	}
	return reply, nil
}

// AddNotice 新建公告信息
func (m *Mysql) AddNotice(request *pb.Notice, extroInfo *pb.MessageExtroInfo) (*pb.Notice, *pb.ErrorMessage) {

	nowTime := time.Now().Unix()
	_, err := m.Db.Exec("insert into "+common.MysqlConfigNotice+
		" (id,title,content,create_time,update_time,hide,order_sort) values(?,?,?,?,?,?,?)",
		request.Id,
		request.Title,
		request.Content,
		nowTime,
		nowTime,
		request.Hide,
		request.OrderSort,
	)
	if err != nil {
		common.LogError("Mysql AddNotice insert has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	return request, nil
}

// UpdNotice 修改公告信息
func (m *Mysql) UpdNotice(request *pb.Notice, extroInfo *pb.MessageExtroInfo) (*pb.Notice, *pb.ErrorMessage) {

	request.UpdateTime = time.Now().Unix()
	_, err := m.Db.Exec("update "+common.MysqlConfigNotice+" set title=?,content=?,update_time=?,hide=?,order_sort=? where id=?",
		&request.Title,
		&request.Content,
		&request.UpdateTime,
		&request.Hide,
		&request.OrderSort,
		&request.Id)

	if err != nil {
		common.LogError("Mysql UpdNotice  has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return request, nil
}

// DelNotice 删除公告信息
func (m *Mysql) DelNotice(request *pb.Notice, extroInfo *pb.MessageExtroInfo) (*pb.Notice, *pb.ErrorMessage) {

	_, err := m.Db.Exec("delete  from "+common.MysqlConfigNotice+"  where id=?",
		&request.Id)

	if err != nil {
		common.LogError("Mysql DelNotice  has err", err)
		return request, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return request, nil
}

// AllianceLeaderRoomCardSettleInfo批量添加副盟主结算信息
func (m *Mysql) AddAllianceLeaderSettleInfo(settleInfo *pb.SaveAllianceLeaderRoomCardSettleInfo, extraInfo *pb.MessageExtroInfo) (*pb.EmptyMessage, *pb.ErrorMessage) {
	reply := &pb.EmptyMessage{}
	if len(settleInfo.Data) == 0 {
		return reply, nil
	}

	sqlStr := `insert into ` + common.MysqlAllianceLeaderRoomCardSettleInfo +
		` (player_uuid,player_shortId,player_account,player_name,real_num,settle_num,commission_percent` +
		` ,commission,start_time,end_time,update_time) values `

	parameters := make([]interface{}, 0)
	now := time.Now().Unix()
	for i := 0; i < len(settleInfo.Data); i++ {
		if i > 0 {
			sqlStr += `,`
		}
		sqlStr += `(?,?,?,?,?,?,?,?,?,?,?)`
		curr := settleInfo.Data[i]
		parameters = append(parameters,
			curr.PlayerUUID,
			curr.PlayerShortId,
			curr.PlayerAccount,
			curr.PlayerName,
			curr.RealRoomCard,
			curr.SettleRoomCard,
			curr.CommissionPercent,
			curr.Commission,
			curr.StartTime,
			curr.EndTime,
			now)
	}

	_, err := m.Db.Exec(sqlStr, parameters...)

	if err != nil {
		common.LogError("Mysql AddAllianceLeaderSettleInfo  has err", err)
		return reply, common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return reply, nil

	/*tx, err := m.Db.Begin()
	if err != nil {
		common.LogError("Mysql AddAllianceLeaderSettleInfo db begin err", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	sqlStr := `insert into ` + common.MysqlAllianceLeaderRoomCardSettleInfo +
		` (player_uuid,player_shortId,player_account,player_name,real_num,settle_num,commission_percent` +
		` ,commission,start_time,end_time,update_time) ` +
		` values (?,?,?,?,?,?,?,?,?,?,?);`

	for _, curr := range settleInfos {
		execItem, err := tx.Prepare(sqlStr)
		if err != nil {
			common.LogError("Mysql AddAllianceLeaderSettleInfo tx.Prepare err", err)
			return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}
		defer execItem.Close()
		if _, err = execItem.Exec(
			curr.PlayerUUID,
			curr.PlayerShortId,
			curr.PlayerAccount,
			curr.PlayerName,
			curr.RealRoomCard,
			curr.SettleRoomCard,
			curr.CommissionPercent,
			curr.Commission,
			curr.StartTime,
			curr.EndTime,
			time.Now().Unix()); err != nil {
			tx.Rollback()
			common.LogError("Mysql AddAllianceLeaderSettleInfo exec err", err)
			return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
		}

	}
	err = tx.Commit()
	if err != nil {
		common.LogError("Mysql AddAllianceLeaderSettleInfo Commit err", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	return nil*/

}
