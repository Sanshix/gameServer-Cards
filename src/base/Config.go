package base

import (
	"database/sql"
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"reflect"
	"time"

	"github.com/golang/protobuf/proto"
)

func init() {
	common.AllComponentMap["Config"] = &Config{}
}

// Config 配置组件
type Config struct {
	common.ConfigI
	Base
	db *sql.DB
	// 全局配置
	cachedGlobalConfig map[string]*pb.GlobalConfig
	// 游戏场次配置
	cachedGameConfig map[pb.GameType]*common.GameSceneMap
	// 礼包配置
	cachedGiftConfig map[string]*pb.GiftInfo
	// 任务配置
	cachedTaskConfig map[string]*pb.TaskConfig
	// 商品配置
	cachedProductConfig map[string]*pb.ProductConfig
	// 跑马灯配置
	cachedHorseRaceLampConfig map[string]*pb.HorseRaceLampConfig
	// 支付配置
	cachedChannelConfig map[string]*pb.ChannelConfig
	// 机器人配置
	cachedRobotActionConfig map[string]*pb.RobotActionConfig
	// 机器人行为组配置
	cachedRobotActionGroupConfig map[string]*pb.RobotActionGroupConfig
}

// Start 开始组件
func (c *Config) Start() {
	//c.DeleteGiftConfig("1")
	//common.LogError(c.GetAllGiftConfig())
}

// LoadComponent 加载组件
func (c *Config) LoadComponent(config *common.OneComponentConfig, componentName string) {
	c.Base.LoadComponent(config, componentName)
	// 全局配置
	c.cachedGlobalConfig = make(map[string]*pb.GlobalConfig)
	// 游戏场次配置
	c.cachedGameConfig = make(map[pb.GameType]*common.GameSceneMap)
	// 礼包配置
	c.cachedGiftConfig = make(map[string]*pb.GiftInfo)
	// 任务配置
	c.cachedTaskConfig = make(map[string]*pb.TaskConfig)
	// 支付配置
	c.cachedChannelConfig = make(map[string]*pb.ChannelConfig)
	// 商品配置
	c.cachedProductConfig = make(map[string]*pb.ProductConfig)
	// 跑马灯配置
	c.cachedHorseRaceLampConfig = make(map[string]*pb.HorseRaceLampConfig)
	// 机器人配置
	c.cachedRobotActionConfig = make(map[string]*pb.RobotActionConfig)
	// 机器人行为组配置
	c.cachedRobotActionGroupConfig = make(map[string]*pb.RobotActionGroupConfig)
	dbTemp, err := sql.Open("mysql", (*c.Config)["connect_string"])
	if err != nil {
		common.LogError("Configer connect mysql err", err)
		panic(err)
	}
	// 设置Conn最大生存时间
	dbTemp.SetConnMaxLifetime(600 * time.Second)
	// 设置空闲连接数0，空闲链接数会被清理
	dbTemp.SetMaxIdleConns(0)
	//dbTemp.SetMaxOpenConns(30)
	c.db = dbTemp
	err = c.checkTable()
	if err != nil {
		common.LogError("Configer LoadComponent err", err)
		panic(err)
	}

	// 加载配置以缓存并每分钟更新缓存
	common.StartTimer(time.Duration(1)*time.Minute, true, func() bool {
		c.getAllGlobalConfigFromDb()
		c.getAllGameConfigFromDb()
		c.getAllGiftConfigFromDb()
		c.getAllTaskConfigFromDb()
		c.getAllChannelConfigFromDb()
		c.getAllProductConfigFromDb()
		c.getAllRobotActionConfigFromDb()
		c.getAllRobotActionGroupConfigFromDb()
		return true
	})
	common.LogInfo("Configer LoadComponent ok")
	return
}

// checkTable 检测表是否存在并创建
func (c *Config) checkTable() error {
	_, err := c.db.Exec(common.MysqlCheckConfigGlobalTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlConfigGlobal has err", err)
		return err
	}

	_, err = c.db.Exec(common.MysqlCheckConfigGamesTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlConfigGames has err", err)
		return err
	}

	_, err = c.db.Exec(common.MysqlCheckConfigChannelsTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlCheckConfigChannelsTable has err", err)
		return err
	}
	_, err = c.db.Exec(common.MysqlCheckConfigGiftsTable)
	if err != nil {
		common.LogError("Mysql checkTable Mysql_Check_Config_Gifts_Table has err", err)
		return err
	}

	_, err = c.db.Exec(common.MysqlCheckConfigTasksTable)
	if err != nil {
		common.LogError("Mysql checkTable Mysql_Check_Config_Tasks_Table has err", err)
		return err
	}
	_, err = c.db.Exec(common.MysqlCheckConfigProductsTable)
	if err != nil {
		common.LogError("Mysql checkTable Mysql_Check_Config_Product_Table has err", err)
		return err
	}
	_, err = c.db.Exec(common.MysqlCheckOrderInfoTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlCheckOrderInfoTable has err", err)
		return err
	}
	_, err = c.db.Exec(common.MysqlConfigHorseRaceLampTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlConfigHorseRaceLampTable has err", err)
		return err
	}
	_, err = c.db.Exec(common.MysqlConfigNoticeTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlConfigNoticeTable has err", err)
		return err
	}
	_, err = c.db.Exec(common.MysqlCheckConfigRobotActionTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlCheckConfigRobotActionTable has err", err)
		return err
	}

	_, err = c.db.Exec(common.MysqlCheckConfigRobotActionGroupTable)
	if err != nil {
		common.LogError("Mysql checkTable MysqlCheckConfigRobotActionGroupTable has err", err)
		return err
	}

	return nil
}

// getGlobalConfigFromDb 根据name从数据库中获取全局配置
func (c *Config) getGlobalConfigFromDb(name string) *pb.GlobalConfig {
	var reply = &pb.GlobalConfig{}
	err := c.db.QueryRow("select * from "+common.MysqlConfigGlobal+" where name=?",
		name,
	).Scan(
		&reply.AutoId,
		&reply.Name,
		&reply.Value,
		&reply.Remark,
	)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select Global Configer from "+common.MysqlConfigGlobal+" has err", err)
		}
		return nil
	}
	return reply
}

// getAllGlobalConfigFromDb 从数据库中获取全部全局配置
func (c *Config) getAllGlobalConfigFromDb() {
	rows, err := c.db.Query("select auto_id,name,value,remark from " + common.MysqlConfigGlobal)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select Global Configer from "+common.MysqlConfigGlobal+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		var out = &pb.GlobalConfig{}
		err = rows.Scan(
			&out.AutoId,
			&out.Name,
			&out.Value,
			&out.Remark)
		if err != nil {
			common.LogError("select Global Configer from "+common.MysqlConfigGlobal+" has err", err)
			continue
		}
		c.cachedGlobalConfig[out.Name] = out
	}
}

//getGameConfigFromDb 根据游戏场次，类型和字段名称获取配置信息
func (c *Config) getGameConfigFromDb(gameScene int32, gameType pb.GameType, name string) *pb.GameConfig {
	var reply = &pb.GameConfig{}
	err := c.db.QueryRow("select auto_id,game_scene,game_type,name,value,remark from "+common.MysqlConfigGames+" where name=? and game_scene=? and game_type=?",
		name,
		gameScene,
		gameType,
	).Scan(
		&reply.AutoId,
		&reply.GameScene,
		&reply.GameType,
		&reply.Name,
		&reply.Value,
		&reply.Remark,
	)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select Game Configer from "+common.MysqlConfigGames+" has err", err)
		}
		return nil
	}
	return reply
}

//getAllGameConfigFromDb 从数据库中获取全部全局配置
func (c *Config) getAllGameConfigFromDb() {
	rows, err := c.db.Query("select auto_id,game_scene,game_type,name,value,remark from " + common.MysqlConfigGames)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select Game Configer from "+common.MysqlConfigGames+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		out := &pb.GameConfig{}
		err = rows.Scan(
			&out.AutoId,
			&out.GameScene,
			&out.GameType,
			&out.Name,
			&out.Value,
			&out.Remark)
		if err != nil {
			common.LogError("select Game Configer from "+common.MysqlConfigGames+" has err", err)
			continue
		}
		c.setData2CachedGameConfig(out)
	}
}

func (c *Config) setData2CachedGameConfig(config *pb.GameConfig) {
	if _, ok := c.cachedGameConfig[config.GameType]; !ok {
		// 新写入配置到内存
		c.cachedGameConfig[config.GameType] = &common.GameSceneMap{
			GameType: config.GameType,
			Map:      make(map[int32]*common.GameKeyMap),
		}
		c.cachedGameConfig[config.GameType].Map[config.GameScene] = &common.GameKeyMap{
			GameScene: config.GameScene,
			GameType:  config.GameType,
			Map:       make(map[string]*pb.GameConfig),
		}
		c.cachedGameConfig[config.GameType].Map[config.GameScene].Map[config.Name] = config

	} else {
		// 更新配置到内存
		sceneMap, ok := c.cachedGameConfig[config.GameType].Map[config.GameScene]
		if !ok {
			sceneMap = &common.GameKeyMap{
				GameScene: config.GameScene,
				GameType:  config.GameType,
				Map:       make(map[string]*pb.GameConfig),
			}
			c.cachedGameConfig[config.GameType].Map[config.GameScene] = sceneMap
		}
		c.cachedGameConfig[config.GameType].Map[config.GameScene].Map[config.Name] = config
	}
}

//checkGlobalNameIsExists 检测全局配置是否存在
func (c *Config) checkGlobalNameIsExists(name string) bool {
	return c.getGlobalConfigFromDb(name) != nil
}

//checkGameNameIsExists 检测游戏配置是否存在
func (c *Config) checkGameNameIsExists(gameScene int32, gameType pb.GameType, name string) bool {
	return c.getGameConfigFromDb(gameScene, gameType, name) != nil
}

//SetGameConfig 设置GameConfig
func (c *Config) SetGameConfig(pb *pb.GameConfig, forceUpdate bool) bool {
	if game, ok := c.cachedGameConfig[pb.GameType]; ok {
		if keyMap, ok := game.Map[pb.GameScene]; ok {
			if _, ok = keyMap.Map[pb.Name]; ok && !forceUpdate {
				return true
			}
		}
	}
	//if name is exists
	if !c.checkGameNameIsExists(pb.GameScene, pb.GameType, pb.Name) {
		//insert config to db
		_, err := c.db.Exec(
			`INSERT INTO `+common.MysqlConfigGames+` (
			game_scene,game_type,name,value,remark)
			VALUES (?,?,?,?,?)`,
			pb.GameScene,
			pb.GameType,
			pb.Name,
			pb.Value,
			pb.Remark)
		if err != nil {
			common.LogError("insert game config err:", err)
			return false
		}

	} else {
		_, err := c.db.Exec(
			`UPDATE `+common.MysqlConfigGames+`
			SET value=?,remark=? where name=? and game_scene=? and game_type=?
			`,
			pb.Value,
			pb.Remark,
			pb.Name,
			pb.GameScene,
			pb.GameType)
		if err != nil {
			common.LogError("update game config err:", err)
			return false
		}
	}
	//update cache
	c.setData2CachedGameConfig(pb)
	return true
}

//SetGameConfigBatch 批量设置GameConfig
func (c *Config) SetGameConfigBatch(pbs []*pb.GameConfig, forceUpdate bool) {
	for i := range pbs {
		c.SetGameConfig(pbs[i], forceUpdate)
	}
}

//GetGameConfigByGameType 获取GameConfig
func (c *Config) GetGameConfigByGameType(gameType pb.GameType) *common.GameSceneMap {
	sceneMap, ok := c.cachedGameConfig[gameType]
	if !ok {
		return nil
	}
	return sceneMap
}

//GetGameConfigByGameTypeAndScene 获取GameConfig
func (c *Config) GetGameConfigByGameTypeAndScene(gameType pb.GameType, gameScene int32) *common.GameKeyMap {
	sceneMap, ok := c.cachedGameConfig[gameType]
	if !ok {
		return nil
	}
	sceneKeyMap, ok := sceneMap.Map[gameScene]
	if !ok {
		return nil
	}
	return sceneKeyMap
}

//GetGameConfig 获取GameConfig
func (c *Config) GetGameConfig(gameType pb.GameType, gameScene int32, name string) *pb.GameConfig {
	sceneMap, ok := c.cachedGameConfig[gameType]
	if !ok {
		return nil
	}
	sceneKeyMap, ok := sceneMap.Map[gameScene]
	if !ok {
		return nil
	}

	if value, ok := sceneKeyMap.Map[name]; ok {
		return value
	}
	return nil
}

// SetGlobal 设置全局配置
func (c *Config) SetGlobal(request *pb.GlobalConfig, forceUpdate bool) bool {
	if _, ok := c.cachedGlobalConfig[request.Name]; ok && !forceUpdate {
		return true
	}
	//if name is exists
	if !c.checkGlobalNameIsExists(request.Name) {
		//insert config to db
		_, err := c.db.Exec(
			`INSERT INTO `+common.MysqlConfigGlobal+` (
			name,value,remark)
			VALUES (?,?,?)`,
			request.Name,
			request.Value,
			request.Remark)
		if err != nil {
			common.LogError("insert global config err:", err)
			return false
		}

	} else {
		_, err := c.db.Exec(
			`UPDATE `+common.MysqlConfigGlobal+`
			SET value=?,remark=? where name=?
			`,
			request.Value,
			request.Remark,
			request.Name)
		if err != nil {
			common.LogError("update global config err:", err)
			return false
		}
	}
	//update cache
	c.cachedGlobalConfig[request.Name] = request
	return true
}

//SetGlobalBatch 批量设置GlobalConfig
func (c *Config) SetGlobalBatch(pbs []*pb.GlobalConfig, forceUpdate bool) {
	for i := range pbs {
		c.SetGlobal(pbs[i], forceUpdate)
	}
}

// GetGlobal 获取全局配置
func (c *Config) GetGlobal(key string) *pb.GlobalConfig {
	if value, ok := c.cachedGlobalConfig[key]; ok {
		return value
	}
	return nil
}

// GetGlobalAll 获得全部全局配置
func (c *Config) GetGlobalAll() []*pb.GlobalConfig {
	arr := make([]*pb.GlobalConfig, 0)
	for key := range c.cachedGlobalConfig {
		item := c.cachedGlobalConfig[key]
		if nil != item {
			arr = append(arr, c.cachedGlobalConfig[key])
		}
	}
	return arr
}

//删除场次
func (c *Config) DeleteGameScene(gameType pb.GameType, gameScene int32) bool {
	//删除数据库中数据
	_, err := c.db.Exec(
		`DELETE FROM `+common.MysqlConfigGames+`
			where game_type=? AND game_scene=?
			`,
		gameType,
		gameScene)
	if err != nil {
		common.LogError("delete game scene err:", err, gameType, gameScene)
		return false
	}

	//删除缓存
	if _, ok := c.cachedGameConfig[gameType]; ok {
		if _, ok = c.cachedGameConfig[gameType].Map[gameScene]; ok {
			delete(c.cachedGameConfig[gameType].Map, gameScene)
		}
	}
	return true
}

//获取数据库里面的所有礼物配置，并填充进config.cachedGiftConfig内存中
func (c *Config) getAllGiftConfigFromDb() {
	rows, err := c.db.Query("select gift_id,content,remark from " + common.MysqlConfigGifts)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select gift Configer1 from "+common.MysqlConfigGifts+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		var gid string
		var contentByte []byte
		var realContent = &pb.GiftInfo{}
		var content = &pb.GiftList{}
		err = rows.Scan(&gid, &contentByte, &realContent.Remark)
		if err != nil {
			common.LogError("select gift Configer2 from "+common.MysqlConfigGifts+" has err", err)
			continue
		}
		err = proto.Unmarshal(contentByte, content)
		if err != nil {
			common.LogError("config getAllGiftConfigFromDb unmarshal has err:", err)
			continue
		}
		realContent.GiftId = gid
		realContent.Content = content
		c.cachedGiftConfig[gid] = realContent
	}

}

//根据id 获取礼物content
func (c *Config) getGiftConfigFromDb(id string) *pb.GiftInfo {
	var replyByte []byte
	var realReply = &pb.GiftInfo{}
	var content = &pb.GiftList{}
	err := c.db.QueryRow("select gift_id,content,remark from "+common.MysqlConfigGifts+" where gift_id=?", id).Scan(
		&realReply.GiftId,
		&replyByte,
		&realReply.Remark,
	)
	if err != nil {
		return nil
	}
	err = proto.Unmarshal(replyByte, content)
	if err != nil {
		common.LogError("Config getGiftConfigFromDb unMarshal has error ", err)
		return nil
	}
	realReply.Content = content
	return realReply
}

//检查数据库里面是否有该礼物id
func (c *Config) checkGiftIdIsExists(id string) bool {
	return c.getGiftConfigFromDb(id) != nil
}

// SetGiftConfig 设置礼物配置
// 思路：模板仅起到方便mysql加载一些通用的配置
//		但一切以内存中的数据为主，即common.Config
//		改变内存数据同时要保存到数据库里面，他又会每60s从数据库查找内容并填充到内存中，为了分布式内存数据统一
func (c *Config) SetGiftConfig(info *pb.GiftInfo, forceUpdate bool) bool {

	// 检查内存有没有礼包id 和 是否强制更新
	if _, ok := c.cachedGiftConfig[info.GiftId]; ok {
		if !forceUpdate {
			return false
		}
	}
	// 只要进入这个步骤就会将模板保存到数据库，想不保存就不要让他进入就行
	// 有这个礼物id,根据配置文件更新，同时保存到内存中
	if c.checkGiftIdIsExists(info.GiftId) {
		//转push
		push, err := proto.Marshal(info.Content)
		_, err = c.db.Exec("update "+common.MysqlConfigGifts+" Set content = ?,remark = ? where gift_id = ?",
			push,
			info.Remark,
			info.GiftId)
		if err != nil {
			common.LogError("Config SetGiftConfig update has err,", err)
			return false
		}
		//没有时，插入该数据
	} else {
		//转push
		push, err := proto.Marshal(info.Content)

		_, err = c.db.Exec("insert into "+common.MysqlConfigGifts+"(gift_id,content,remark) value (?,?,?)",
			info.GiftId,
			push,
			info.Remark,
		)
		if err != nil {
			common.LogError("Config SetGiftConfig insert has err,", err)
			return false
		}
	}
	//保存到内存
	c.cachedGiftConfig[info.GiftId] = info
	return true
}

//GetGiftConfig 获取礼物配置
//参数：礼物的id
//返回：礼包内容giftInfo
func (c *Config) GetGiftConfig(id string) *pb.GiftInfo {
	if Content, ok := c.cachedGiftConfig[id]; ok {
		return Content
	}
	return nil
}

//GetAllGiftConfig 获取全部礼物配置
//返回：全部礼包内容giftInfo数组
func (c *Config) GetAllGiftConfig() []*pb.GiftInfo {
	reply := make([]*pb.GiftInfo, 0)
	for _, k := range c.cachedGiftConfig {
		reply = append(reply, k)
	}
	return reply
}

//DeleteGiftConfig 删除礼物配置
//参数：礼物的id
//返回：是否成功
func (c *Config) DeleteGiftConfig(id string) bool {
	//删除数据库
	_, err := c.db.Exec(
		`DELETE FROM `+common.MysqlConfigGifts+`
			where gift_id =?
			`,
		id)
	if err != nil {
		common.LogError("Config DeleteGiftConfig exec err:", err, id)
		return false
	}
	//删除内存
	if _, ok := c.cachedGiftConfig[id]; ok {
		delete(c.cachedGiftConfig, id)
		return true
	}
	return false
}

///////////////////////任务配置开始////////////////////////
//获取数据库里面的所有任务配置，并填充进config.cachedTaskConfig内存中
func (c *Config) getAllTaskConfigFromDb() {
	rows, err := c.db.Query("select uuid,content from " + common.MysqlConfigTasks)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select task Configer1 from "+common.MysqlConfigTasks+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		var uuid string
		var contentByte []byte
		var content = &pb.TaskConfig{}
		err = rows.Scan(&uuid, &contentByte)
		if err != nil {
			common.LogError("select task Configer2 from "+common.MysqlConfigTasks+" has err", err)
			continue
		}
		err = proto.Unmarshal(contentByte, content)
		if err != nil {
			common.LogError("config getAllTaskConfigFromDb unmarshal has err:", err)
			continue
		}
		c.cachedTaskConfig[uuid] = content
	}

}

// AddTaskConfig 增加任务配置
func (c *Config) AddTaskConfig(info *pb.TaskConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("insert into "+common.MysqlConfigTasks+"(uuid,content) value (?,?)",
		info.GetTaskUuid(),
		push,
	)
	if err != nil {
		common.LogError("Config AddTaskConfig insert has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	//保存到内存
	c.cachedTaskConfig[info.GetTaskUuid()] = info
	return nil
}

// UpdateTaskConfig 修改任务配置
func (c *Config) UpdateTaskConfig(info *pb.TaskConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("update "+common.MysqlConfigTasks+" Set content = ? where uuid = ?",
		push,
		info.GetTaskUuid())
	if err != nil {
		common.LogError("Config UpdateTaskConfig update has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//保存到内存
	c.cachedTaskConfig[info.GetTaskUuid()] = info
	return nil
}

//GetTaskConfig 获取任务配置
func (c *Config) GetTaskConfig(uuid string) *pb.TaskConfig {
	if Content, ok := c.cachedTaskConfig[uuid]; ok {
		return Content
	}
	return nil
}

//GetAllTaskConfig 获取全部礼物配置
func (c *Config) GetAllTaskConfig() []*pb.TaskConfig {
	reply := make([]*pb.TaskConfig, 0)
	for _, k := range c.cachedTaskConfig {
		reply = append(reply, k)
	}
	return reply
}

//DeleteTaskConfig 删除任务配置
func (c *Config) DeleteTaskConfig(uuid string) *pb.ErrorMessage {
	//删除数据库
	_, err := c.db.Exec(
		`DELETE FROM `+common.MysqlConfigTasks+`
			where uuid =?
			`,
		uuid)
	if err != nil {
		common.LogError("Config DeleteTaskConfig exec err:", err, uuid)
	}
	//删除内存
	if _, ok := c.cachedTaskConfig[uuid]; ok {
		delete(c.cachedTaskConfig, uuid)
	}
	return nil
}

//////////////////////任务配置结束///////////////////////////////

///////////////////////商品配置开始////////////////////////
//获取数据库里面的所有商品配置，并填充进config.cachedProductConfig内存中
func (c *Config) getAllProductConfigFromDb() {
	rows, err := c.db.Query("select uuid,content from " + common.MysqlConfigProducts)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select product Configer from "+common.MysqlConfigProducts+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		var uuid string
		var contentByte []byte
		var content = &pb.ProductConfig{}
		err = rows.Scan(&uuid, &contentByte)
		if err != nil {
			common.LogError("select Product Configer2 from "+common.MysqlConfigProducts+" has err", err)
			continue
		}
		err = proto.Unmarshal(contentByte, content)
		if err != nil {
			common.LogError("config getAllProductConfigFromDb unmarshal has err:", err)
			continue
		}
		c.cachedProductConfig[uuid] = content
	}

}

// AddProductConfig 增加商品配置
func (c *Config) AddProductConfig(info *pb.ProductConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("insert into "+common.MysqlConfigProducts+"(uuid,content) value (?,?)",
		info.GetProductUuid(),
		push,
	)
	if err != nil {
		common.LogError("Config AddProductConfig insert has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	//保存到内存
	c.cachedProductConfig[info.GetProductUuid()] = info
	return nil
}

// UpdateProductConfig 修改商品配置
func (c *Config) UpdateProductConfig(info *pb.ProductConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("update "+common.MysqlConfigProducts+" Set content = ? where uuid = ?",
		push,
		info.GetProductUuid())
	if err != nil {
		common.LogError("Config UpdateTaskConfig update has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//保存到内存
	c.cachedProductConfig[info.GetProductUuid()] = info
	return nil
}

//GetProductConfig 获取商品配置
func (c *Config) GetProductConfig(uuid string) *pb.ProductConfig {
	if Content, ok := c.cachedProductConfig[uuid]; ok {
		return Content
	}
	return nil
}

//GetAllProductConfig 获取全部商品配置
func (c *Config) GetAllProductConfig() []*pb.ProductConfig {
	reply := make([]*pb.ProductConfig, 0)
	for _, k := range c.cachedProductConfig {
		reply = append(reply, k)
	}
	//按原价格升序排序
	swap := reflect.Swapper(reply)
	for i := 0; i < len(reply); i++ {
		for j := i + 1; j < len(reply); j++ {
			if reply[i].OriginalPrice > reply[j].OriginalPrice {
				swap(i, j)
			}
		}
	}

	return reply
}

//DeleteProductConfig 删除商品配置
func (c *Config) DeleteProductConfig(uuid string) *pb.ErrorMessage {
	//删除数据库
	_, err := c.db.Exec(
		`DELETE FROM `+common.MysqlConfigProducts+`
			where uuid =?
			`,
		uuid)
	if err != nil {
		common.LogError("Config DeleteProductConfig exec err:", err, uuid)
	}
	//删除内存
	if _, ok := c.cachedProductConfig[uuid]; ok {
		delete(c.cachedProductConfig, uuid)
	}
	return nil
}

//////////////////////商品配置结束///////////////////////////////

///////////////////////跑马灯配置开始////////////////////////
//获取数据库里面的所有跑马灯配置，并填充进config.cachedHorseRaceLampConfig内存中
func (c *Config) getAllHorseRaceLampConfigFromDb() {
	rows, err := c.db.Query("select uuid,content from " +
		common.MysqlConfigHorseRaceLamp)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select HorseRaceLamp Configer from "+common.MysqlConfigHorseRaceLamp+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		var uuid string
		var contentByte []byte
		var content = &pb.HorseRaceLampConfig{}
		err = rows.Scan(&uuid, &contentByte)
		if err != nil {
			common.LogError("select HorseRaceLamp Configer2 from "+common.MysqlConfigHorseRaceLamp+" has err", err)
			continue
		}
		err = proto.Unmarshal(contentByte, content)
		if err != nil {
			common.LogError("config getAllHorseRaceLampConfigFromDb unmarshal has err:", err)
			continue
		}
		c.cachedHorseRaceLampConfig[uuid] = content
	}

}

// AddHorseRaceLampConfig 增加跑马灯配置
func (c *Config) AddHorseRaceLampConfig(info *pb.HorseRaceLampConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("insert into "+common.MysqlConfigHorseRaceLamp+"(uuid,content) value (?,?)",
		info.GetUuid(),
		push,
	)
	if err != nil {
		common.LogError("Config AddHorseRaceLampConfig insert has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	//保存到内存
	c.cachedHorseRaceLampConfig[info.GetUuid()] = info
	return nil
}

// UpdateHorseRaceLampConfig 修改跑马灯配置
func (c *Config) UpdateHorseRaceLampConfig(info *pb.HorseRaceLampConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("update "+common.MysqlConfigHorseRaceLamp+" Set content = ? where uuid = ?",
		push,
		info.GetUuid())
	if err != nil {
		common.LogError("Config UpdateTaskConfig update has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//保存到内存
	c.cachedHorseRaceLampConfig[info.GetUuid()] = info
	return nil
}

//GetHorseRaceLampConfig 获取跑马灯配置
func (c *Config) GetHorseRaceLampConfig(uuid string) *pb.HorseRaceLampConfig {
	if Content, ok := c.cachedHorseRaceLampConfig[uuid]; ok {
		return Content
	}
	return nil
}

//GetAllHorseRaceLampConfig 获取全部跑马灯配置
func (c *Config) GetAllHorseRaceLampConfig() []*pb.HorseRaceLampConfig {
	reply := make([]*pb.HorseRaceLampConfig, 0)
	for _, k := range c.cachedHorseRaceLampConfig {
		reply = append(reply, k)
	}

	//按时间排序
	swap := reflect.Swapper(reply)
	for i := 0; i < len(reply); i++ {
		for j := i + 1; j < len(reply); j++ {
			if reply[i].UpdateTime > reply[j].UpdateTime {
				swap(i, j)
			}
		}
	}

	return reply
}

//DeleteHorseRaceLampConfig 删除跑马灯配置
func (c *Config) DeleteHorseRaceLampConfig(uuid string) *pb.ErrorMessage {
	//删除数据库
	_, err := c.db.Exec(
		`DELETE FROM `+common.MysqlConfigHorseRaceLamp+`
			where uuid =?
			`,
		uuid)
	if err != nil {
		common.LogError("Config MysqlConfigHorseRaceLamp exec err:", err, uuid)
	}
	//删除内存
	if _, ok := c.cachedHorseRaceLampConfig[uuid]; ok {
		delete(c.cachedHorseRaceLampConfig, uuid)
	}
	return nil
}

//////////////////////跑马灯配置结束///////////////////////////////

///////////////////////支付通道配置开始////////////////////////
//获取数据库里面的所有支付通道配置，并填充进config.cachedChannelConfig内存中
func (c *Config) getAllChannelConfigFromDb() {
	rows, err := c.db.Query("select uuid,content from " + common.MysqlConfigChannel)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select Channel Configer from "+common.MysqlConfigChannel+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		var uuid string
		var contentByte []byte
		var content = &pb.ChannelConfig{}
		err = rows.Scan(&uuid, &contentByte)
		if err != nil {
			common.LogError("select Channel Configer2 from "+common.MysqlConfigChannel+" has err", err)
			continue
		}
		err = proto.Unmarshal(contentByte, content)
		if err != nil {
			common.LogError("config getAllProductConfigFromDb unmarshal has err:", err)
			continue
		}
		c.cachedChannelConfig[uuid] = content
	}

}

// AddChannelConfig 增加支付通道配置
func (c *Config) AddChannelConfig(info *pb.ChannelConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("insert into "+common.MysqlConfigChannel+"(uuid,content) value (?,?)",
		info.GetChannelUuid(),
		push,
	)
	if err != nil {
		common.LogError("Config AddProductConfig insert has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	//保存到内存
	c.cachedChannelConfig[info.GetChannelUuid()] = info
	return nil
}

// UpdateChannelConfig 修改支付通道配置
func (c *Config) UpdateChannelConfig(info *pb.ChannelConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("update "+common.MysqlConfigChannel+" Set content = ? where uuid = ?",
		push,
		info.ChannelUuid)
	if err != nil {
		common.LogError("Config UpdateChannelConfig update has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//保存到内存
	c.cachedChannelConfig[info.GetChannelUuid()] = info
	return nil
}

//GetChannelConfig 获取支付通道配置
func (c *Config) GetChannelConfig(uuid string) *pb.ChannelConfig {
	if Content, ok := c.cachedChannelConfig[uuid]; ok {
		return Content
	}
	return nil
}

//GetAllChannelConfig 获取全部支付通道配置
func (c *Config) GetAllChannelConfig() []*pb.ChannelConfig {
	reply := make([]*pb.ChannelConfig, 0)
	for _, k := range c.cachedChannelConfig {
		reply = append(reply, k)
	}
	return reply
}

//DeleteChannelConfig 删除支付通道配置
func (c *Config) DeleteChannelConfig(uuid string) *pb.ErrorMessage {
	//删除数据库
	_, err := c.db.Exec(
		`DELETE FROM `+common.MysqlConfigChannel+`
			where uuid =?
			`,
		uuid)
	if err != nil {
		common.LogError("Config DeleteProductConfig exec err:", err, uuid)
	}
	//删除内存
	if _, ok := c.cachedChannelConfig[uuid]; ok {
		delete(c.cachedChannelConfig, uuid)
	}
	return nil
}

//////////////////////支付通道配置结束///////////////////////////////

///////////////////////机器人行为组配置开始////////////////////////
//获取数据库里面的所有机器人行为组配置，并填充进内存中
func (c *Config) getAllRobotActionGroupConfigFromDb() {
	rows, err := c.db.Query("select uuid,content from " + common.MysqlConfigRobotActionGroup)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select robot action group Configer1 from "+common.MysqlConfigRobotActionGroup+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		var uuid string
		var contentByte []byte
		var content = &pb.RobotActionGroupConfig{}
		err = rows.Scan(&uuid, &contentByte)
		if err != nil {
			common.LogError("select robot action group Configer2 from "+common.MysqlConfigRobotActionGroup+" has err", err)
			continue
		}
		err = proto.Unmarshal(contentByte, content)
		if err != nil {
			common.LogError("config getAllRobotActionGroupConfigFromDb unmarshal has err:", err)
			continue
		}
		c.cachedRobotActionGroupConfig[uuid] = content
	}

}

// AddRobotActionGroupConfig 增加机器人行为组配置
func (c *Config) AddRobotActionGroupConfig(info *pb.RobotActionGroupConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("insert into "+common.MysqlConfigRobotActionGroup+"(uuid,content) value (?,?)",
		info.GetActionGroupUuid(),
		push,
	)
	if err != nil {
		common.LogError("Config AddRobotActionGroupConfig insert has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	//保存到内存
	c.cachedRobotActionGroupConfig[info.GetActionGroupUuid()] = info
	return nil
}

// UpdateRobotActionGroupConfig 修改机器人行为组配置
func (c *Config) UpdateRobotActionGroupConfig(info *pb.RobotActionGroupConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("update "+common.MysqlConfigRobotActionGroup+" Set content = ? where uuid = ?",
		push,
		info.GetActionGroupUuid())
	if err != nil {
		common.LogError("Config UpdateRobotActionGroupConfig update has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//保存到内存
	c.cachedRobotActionGroupConfig[info.GetActionGroupUuid()] = info
	return nil
}

//GetRobotActionGroupConfig 获取机器人行为组配置
func (c *Config) GetRobotActionGroupConfig(uuid string) *pb.RobotActionGroupConfig {
	if Content, ok := c.cachedRobotActionGroupConfig[uuid]; ok {
		return Content
	}
	return nil
}

//GetAllRobotActionGroupConfig 获取全部机器人行为组配置
func (c *Config) GetAllRobotActionGroupConfig() []*pb.RobotActionGroupConfig {
	reply := make([]*pb.RobotActionGroupConfig, 0)
	for _, k := range c.cachedRobotActionGroupConfig {
		reply = append(reply, k)
	}
	return reply
}

//////////////////////机器人行为组配置结束///////////////////////////////

///////////////////////机器人行为配置开始////////////////////////
//获取数据库里面的所有机器人行为配置，并填充进内存中
func (c *Config) getAllRobotActionConfigFromDb() {
	rows, err := c.db.Query("select uuid,content from " + common.MysqlConfigRobotAction)
	if err != nil {
		if err != sql.ErrNoRows {
			common.LogError("select robot action Configer1 from "+common.MysqlConfigRobotAction+" has err", err)
		}
		return
	}

	for {
		if !rows.Next() {
			break
		}
		var uuid string
		var contentByte []byte
		var content = &pb.RobotActionConfig{}
		err = rows.Scan(&uuid, &contentByte)
		if err != nil {
			common.LogError("select robot action Configer2 from "+common.MysqlConfigRobotAction+" has err", err)
			continue
		}
		err = proto.Unmarshal(contentByte, content)
		if err != nil {
			common.LogError("config getAllRobotActionConfigFromDb unmarshal has err:", err)
			continue
		}
		c.cachedRobotActionConfig[uuid] = content
	}

}

// AddRobotActionConfig 增加机器人行为配置
func (c *Config) AddRobotActionConfig(info *pb.RobotActionConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("insert into "+common.MysqlConfigRobotAction+"(uuid,content) value (?,?)",
		info.GetActionUuid(),
		push,
	)
	if err != nil {
		common.LogError("Config AddRobotActionConfig insert has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}

	//保存到内存
	c.cachedRobotActionConfig[info.GetActionUuid()] = info
	return nil
}

// UpdateRobotActionConfig 修改机器人行为配置
func (c *Config) UpdateRobotActionConfig(info *pb.RobotActionConfig) *pb.ErrorMessage {
	push, err := proto.Marshal(info)
	_, err = c.db.Exec("update "+common.MysqlConfigRobotAction+" Set content = ? where uuid = ?",
		push,
		info.GetActionUuid())
	if err != nil {
		common.LogError("Config UpdateRobotActionConfig update has err,", err)
		return common.GetGrpcErrorMessage(pb.ErrorCode_ServerError, "")
	}
	//保存到内存
	c.cachedRobotActionConfig[info.GetActionUuid()] = info
	return nil
}

//GetRobotActionConfig 获取机器人行为配置
func (c *Config) GetRobotActionConfig(uuid string) *pb.RobotActionConfig {
	if Content, ok := c.cachedRobotActionConfig[uuid]; ok {
		return Content
	}
	return nil
}

//GetAllRobotActionConfig 获取全部机器人行为配置
func (c *Config) GetAllRobotActionConfig() []*pb.RobotActionConfig {
	reply := make([]*pb.RobotActionConfig, 0)
	for _, k := range c.cachedRobotActionConfig {
		reply = append(reply, k)
	}
	return reply
}

//////////////////////机器人行为配置结束///////////////////////////////
