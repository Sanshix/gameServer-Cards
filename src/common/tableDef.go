package common

const (
	// PlayerMsgQueue 玩家推送消息的队列名称，以玩家uuid为区分
	PlayerMsgQueue string = "queue:player:"
	// ReportMsgQueue 报表队列
	ReportMsgQueue string = "queue:report"
	//PlayerStartPlayGameExchange 玩家开始游戏消息交换机
	PlayerStartPlayGameExchange string = "exchange:player:start-play"

	//PlayerStartPlayGameExchangeKey 玩家开始游戏消息 player.gameStart.[gameType].[roomType].[roomId].[roundId]
	PlayerStartPlayGameExchangeKey string = "player.gameStart."
)
const (
	RedisTokenMap string = "hash:token"
	// RedisOnlineUserTable 玩家在线信息表
	RedisOnlineUserTable string = "hash:online"
	// RedisPlayerInfoTable 玩家信息表
	RedisPlayerInfoTable string = "string:player"
	// RedisShortidTable 短位id表
	RedisShortidTable string = "set:shortid"
	// RedisRoomsTable 房间信息表 (key:gametype,serverIndex)
	RedisRoomsTable string = "hash:rooms:"

	//RedisLeaderBoardNabob 富豪排行榜表
	RedisLeaderBoardNabob string = "zset:leaderboard:nabob"
	//RedisLeaderBoardBigWinner 大赢家排行榜表
	RedisLeaderBoardBigWinner string = "zset:leaderboard:bigwinner"
	// RedisSpinRecordTable 转盘获奖记录表
	RedisSpinRecordTable string = "list:spin:record"
	// RedisRobotManagerInfoTable 机器人管理信息表
	RedisRobotManagerInfoTable string = "string:robot:manager:info"
	// RedisSmsCaptchaTable 短信验证码表
	RedisSmsCaptchaTable string = "set:sms:captcha:"
	// RedisSmsCaptchaIntervalTimeTable 短信验证码请求间隔表
	RedisSmsCaptchaIntervalTimeTable string = "set:sms:captcha:interval:"

	// -----------------------------------------------------
	// 俱乐部Redis字段Start
	// -----------------------------------------------------

	// RedisClubCreateTodayTable 用户今日创建的俱乐部数量统计，在凌晨过期
	RedisClubCreateTodayTable string = "hash:club:create:today"
	// RedisClubInviteCodeTable 俱乐部邀请码表（7位邀请码）
	RedisClubInviteCodeTable string = "set:club:invitecode"
	// RedisHashClubIvitecodeUUIDTable 使用邀请码查询俱乐部UUID的哈希表

	// -----------------------------------------------------
	// 俱乐部Redis字段End
	// -----------------------------------------------------

	// -----------------------------------------------------
	// 联盟Redis字段Start
	// -----------------------------------------------------

	//RedisLeagueMemberIdsTables 记录联盟成员id
	RedisLeagueMemberIdsTable string = "set:league:member:ids"
	// RedisLeagueInviteCodeTable 联盟邀请码表（7位邀请码）
	RedisLeagueInviteCodeTable string = "set:league:invitecode"
	// RedisLeagueApplyIDCount 加入联盟的申请ID计数
	RedisLeagueApplyIDCount string = "count:league:applyid"
	// 联盟Redis字段End
	// -----------------------------------------------------



	// -----------------------------------------------------
	// 代理Redis字段End
	// -----------------------------------------------------

	// RedisBonusTable 奖池表
	RedisBonusTable string = "string:bonus"
	// RedisBonusRobotTable 机器人奖池表
	RedisBonusRobotTable string = "string:bonus:robot"
	// RedisBonusSystemTable 系统奖池表
	RedisBonusSystemTable string = "string:bonus:system"
	// RedisBonusConfigTable 奖池爆率配置表
	RedisBonusConfigTable string = "hash:bonus:config"

	// RedisRoomCodeInfoTable 房间码信息表
	RedisRoomCodeInfoTable string = "hash:roomCode:info"

	// RedisBloodSlotTable 血池信息表
	RedisBloodSlotTable string = "hash:bloodSlot"

	// -----------------------------------------------------
	// 幸运宝石版本Start
	// -----------------------------------------------------

	// RedisMiningProgressTable 挖矿进度表
	RedisMiningProgressTable string = "hash:miningProgress"

	// -----------------------------------------------------
	// 幸运宝石版本End
	// -----------------------------------------------------
)

const (
	// MysqlAccountInfoTable 账号信息表
	MysqlAccountInfoTable string = "game_account_info"
	// MysqlPlayerInfoTable 玩家信息表
	MysqlPlayerInfoTable string = "game_player_info"
	// MysqlConfigGlobal 全局配置表
	MysqlConfigGlobal string = "config_global"
	// MysqlConfigGames 游戏配置表
	MysqlConfigGames string = "config_games"
	// MysqlConfigGifts 游戏配置表
	MysqlConfigGifts string = "config_gifts"
	// MysqlConfigTasks 任务配置表
	MysqlConfigTasks string = "config_tasks"
	// MysqlConfigProducts 商品配置表
	MysqlConfigProducts string = "config_products"
	// MysqlConfigChannel 支付通道配置表
	MysqlConfigChannel string = "config_channel"
	// MysqlClubInfoTable 俱乐部信息表
	MysqlClubInfoTable string = "game_club_info"
	/*// MysqlLeagueInfoTable 联盟信息表
	MysqlLeagueInfoTable string = "league_info"
	// MysqlLeagueMemberTable 联盟成员表
	MysqlLeagueMemberTable string = "league_member"
	// MysqlLeaguePartnerTable 联盟合伙人表
	MysqlLeaguePartnerTable string = "league_partner"
	// MysqlLeagueClubTable 联盟俱乐部表
	MysqlLeagueClubTable string = "league_club"*/
	// MysqlConfigRobotAction 机器人行为配置表
	MysqlConfigRobotAction string = "config_robot_actions"
	// MysqlConfigNotice 公告表
	MysqlConfigNotice string = "config_notice"
	// HorseRaceLamp 跑马灯表
	MysqlConfigHorseRaceLamp string = "config_horseracelamp"
	// MysqlLeagueInfoTable 大联盟信息表
	MysqlLeagueInfoTable string = "game_league_info"

	// MysqlConfigRobotActionGroup 机器人行为组配置表
	MysqlConfigRobotActionGroup string = "config_robot_actions_group"
	// MysqlConfigSplitTableInfo 分表信息表
	MysqlConfigSplitTableInfo string = "config_split_table_info"
	// MysqlReportGameRecord 游戏记录报表
	MysqlReportGameRecord string = "report_game_record"
	// MysqlReportBalanceChangeRecord 用户金额变动报表
	MysqlReportBalanceChangeRecord string = "report_balance_change_record"
	// MysqlReportBeanChangeRecord 用户金额变动报表
	MysqlReportBeanChangeRecord string = "report_bean_change_record"
	// MysqlReportGiveAwayRecord 玩家赠送记录报表
	MysqlReportGiveAwayRecord string = "report_giveaway_record"
	// MysqlConfigAgent 代理配置表
	MysqlConfigAgent string = "config_agent"
	//MysqlOrderInfoTable 订单表
	MysqlOrderInfoTable string = "game_order_info"
	// MysqlReportAchievementChangeRecord 用户业绩记录变动报表
	MysqlReportAchievementChangeRecord string = "report_achievement_change_record"
	// MysqlReportGetCommissionRecord 用户佣金提取报表
	MysqlReportGetCommissionRecord string = "report_get_commission_record"
	// MysqlReportLeagueCommissionRecord 大联盟代理佣金表
	MysqlReportLeagueCommissionRecord string = "report_league_commission_record"
	// MysqlReportRoomCardChangeRecord 房卡变动记录
	MysqlReportRoomCardChangeRecord string = "report_room_card_change_record"
	// MysqlReportBonusRecord 奖池变动记录
	MysqlReportBonusRecord string = "report_bonus_change_record"
	// MysqlAllianceLeaderRoomCardSettleInfo 副盟主结算信息
	MysqlAllianceLeaderRoomCardSettleInfo string = "report_alliance_leader_room_card_settle_info"
)

const (
	//MysqlConfigNotice 公告表sql
	MysqlConfigNoticeTable = `CREATE TABLE IF NOT EXISTS config_notice (
   id bigint(20) NOT NULL AUTO_INCREMENT,
  title varchar(255) DEFAULT NULL COMMENT '公告标题',
  content varchar(255) DEFAULT NULL COMMENT '标题里的内容',
  create_time bigint(20) DEFAULT NULL,
  update_time bigint(20) DEFAULT NULL,
  hide int(1) DEFAULT '0' COMMENT '是否隐藏0:否 1:是',
  order_sort bigint(20) DEFAULT NULL COMMENT '公告展示排名顺序',
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=REDUNDANT;`

	// MysqlConfigNoticeTable 跑马灯表sql
	MysqlConfigHorseRaceLampTable string = `CREATE TABLE IF NOT EXISTS config_horseracelamp (
  id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  uuid varchar(255) NOT NULL,
  content mediumblob NOT NULL,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 ROW_FORMAT=REDUNDANT;`

	// MysqlCheckOrderInfoTable 玩家订单表建表sql
	MysqlCheckOrderInfoTable string = `
		CREATE TABLE IF NOT EXISTS game_order_info (
		auto_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT  COMMENT '订单自增ID',
		uuid VARCHAR(128) NOT NULL PRIMARY KEY,
		player_uuid VARCHAR(128) NOT NULL COMMENT '玩家UUID' ,
		txn_id  VARCHAR(128)  COMMENT '上游订单号',
		order_amount bigint(64) not null COMMENT '订单金额',
		order_state VARCHAR(2) not null COMMENT '订单状态',
		product_uuid VARCHAR(128) not null COMMENT '商品ID',
		product_type  VARCHAR(10) not null COMMENT '商品类型',
		product_num  BIGINT not null COMMENT '商品数量',
		currency  BIGINT not null COMMENT '订单币种',
		channel VARCHAR(10) not null COMMENT '支付通道',
		create_time BIGINT NOT NULL,
		update_time BIGINT NOT NULL,
		remark VARCHAR(2000),
		UNIQUE KEY (auto_id),
		UNIQUE KEY (uuid)
	) 
	ENGINE=InnoDB	 
	DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci
	AUTO_INCREMENT=1;
	`

	// MysqlCheckPlayerInfoTable 玩家数据表建表sql
	MysqlCheckPlayerInfoTable string = `
		CREATE TABLE IF NOT EXISTS game_player_info (
		auto_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		uuid VARCHAR(128) NOT NULL PRIMARY KEY,
		short_id VARCHAR(10) NOT NULL,
		info MEDIUMBLOB NOT NULL,
		update_time BIGINT NOT NULL,
		KEY (update_time),
		UNIQUE KEY (auto_id),
		UNIQUE KEY (short_id)
	) 
	ENGINE=InnoDB	 
	DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci
	AUTO_INCREMENT=1;
	`
	// MysqlCheckAccountInfoTable 账号表建表sql
	MysqlCheckAccountInfoTable string = `
		CREATE TABLE IF NOT EXISTS game_account_info (
		auto_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		uuid VARCHAR(128) NOT NULL PRIMARY KEY,
		short_id VARCHAR(10) NOT NULL,
		account VARCHAR(128) NOT NULL,
		password VARCHAR(256) NOT NULL,
		role_type INT NOT NULL,
		source_type INT NOT NULL,
		open_id varchar(128) null,
		union_id varchar(128) null,
		mobile varchar(30) null,
		create_time bigint not null,
		update_time BIGINT NOT NULL,
		KEY (update_time),
		UNIQUE KEY (account),
		UNIQUE KEY (auto_id),
		KEY (password),
		UNIQUE KEY (short_id),
		UNIQUE KEY (mobile)
	) 
	ENGINE=InnoDB	 
	DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci
	AUTO_INCREMENT=1;
	`
	// MysqlCheckConfigGlobalTable 全局配置表建表sql
	MysqlCheckConfigGlobalTable string = `
		CREATE TABLE IF NOT EXISTS config_global(
		auto_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		name VARCHAR(256) NOT NULL,
		value VARCHAR(1024) NULL,
		remark VARCHAR(256) NULL,
		UNIQUE KEY (auto_id)
	) 
	ENGINE=InnoDB	 
	DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci
	AUTO_INCREMENT=1;
	`
	// MysqlCheckConfigGamesTable 游戏配置表建表sql
	MysqlCheckConfigGamesTable string = `
		CREATE TABLE IF NOT EXISTS config_games(
		auto_id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
		game_scene INT NOT NULL,
		game_type INT NOT NULL,
		name VARCHAR(256) NOT NULL,
		value VARCHAR(1024) NULL,
		remark VARCHAR(256) NULL,
		UNIQUE KEY (auto_id)
		)
		ENGINE=InnoDB	 
		DEFAULT CHARACTER SET=utf8mb4 COLLATE=utf8mb4_general_ci
		AUTO_INCREMENT=1;
	`
	// MysqlCheckConfigGiftsTable 礼包表建表sql
	MysqlCheckConfigGiftsTable string = `
	CREATE TABLE IF NOT EXISTS config_gifts (
		auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
		gift_id VARCHAR(128) NOT NULL,
		content MEDIUMBLOB NOT NULL,
		remark VARCHAR(256) NOT NULL DEFAULT "",
		UNIQUE KEY auto_id (auto_id) USING BTREE,
		UNIQUE KEY (gift_id) USING BTREE
	) ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
	`
	// MysqlCheckConfigTasksTable 任务表建表sql
	MysqlCheckConfigTasksTable string = `
	CREATE TABLE IF NOT EXISTS config_tasks (
		auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
		uuid VARCHAR(128) NOT NULL,
		content MEDIUMBLOB NOT NULL,
		UNIQUE KEY auto_id (auto_id) USING BTREE,
		UNIQUE KEY (uuid) USING BTREE
	) ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
	`

	// MysqlCheckConfigProductsTable 商品表建表sql
	MysqlCheckConfigProductsTable string = `
	CREATE TABLE IF NOT EXISTS config_products (
		auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
		uuid VARCHAR(128) NOT NULL,
		content MEDIUMBLOB NOT NULL,
		UNIQUE KEY auto_id (auto_id) USING BTREE,
		UNIQUE KEY (uuid) USING BTREE
	) ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
	`
	// MysqlCheckConfigChannelsTable 支付通道表建表sql
	MysqlCheckConfigChannelsTable string = `
	CREATE TABLE IF NOT EXISTS config_channel (
		auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
		uuid VARCHAR(128) NOT NULL,
		content MEDIUMBLOB NOT NULL,
		UNIQUE KEY auto_id (auto_id) USING BTREE,
		UNIQUE KEY (uuid) USING BTREE
	) ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
	`

	// MysqlCheckConfigRobotActionTable 任务表建表sql
	MysqlCheckConfigRobotActionTable string = `
	CREATE TABLE IF NOT EXISTS config_robot_actions (
		auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
		uuid VARCHAR(128) NOT NULL,
		content MEDIUMBLOB NOT NULL,
		UNIQUE KEY auto_id (auto_id) USING BTREE,
		UNIQUE KEY (uuid) USING BTREE
	) ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
	`

	// MysqlCheckConfigRobotActionGroupTable 任务表建表sql
	MysqlCheckConfigRobotActionGroupTable string = `
	CREATE TABLE IF NOT EXISTS config_robot_actions_group (
		auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
		uuid VARCHAR(128) NOT NULL,
		content MEDIUMBLOB NOT NULL,
		UNIQUE KEY auto_id (auto_id) USING BTREE,
		UNIQUE KEY (uuid) USING BTREE
	) ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
	`

	// MysqlCheckConfigSplitTableInfo 分表信息表
	MysqlCheckConfigSplitTableInfo string = `
	CREATE TABLE IF NOT EXISTS config_split_table_info(
		auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
		table_name VARCHAR(50) NOT NULL,
		split_name VARCHAR(50) NOT NULL,
		start_timestamp BIGINT(20) NOT NULL,
		end_timestamp BIGINT(20) NOT NULL,
		created_timestamp BIGINT(20) NOT NULL,
		UNIQUE KEY auto_id (auto_id) USING BTREE
	) ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
	`

	// MysqlCheckReportGameRecord 用户游戏记录报表
	MysqlCheckReportGameRecord string = `
	CREATE TABLE IF NOT EXISTS report_game_record(
	auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
	room_id VARCHAR(128) NOT NULL DEFAULT '',
	round_id VARCHAR(128) NOT NULL DEFAULT '',
	game_type int(11) NOT NULL,
	game_scene int(11) NOT NULL,
	game_mode int(11) NOT NULL,
	room_type int(11) NOT NULL,
	player_uuid varchar(128) NOT NULL,
	player_shortId varchar(10) NOT NULL,
	player_account varchar(128) NOT NULL,
	start_time bigint(20) not null,
	settle_time bigint(20) not null,
	before_balance bigint(20) not null,
	total_bet bigint(20) not null,
	win_or_lose bigint(20) not null,
	settle_balance bigint(20) not null,
	commission bigint(20) not null default 0 COMMENT '抽水',
	jackpot_commission bigint(20) not null default 0 COMMENT '奖池抽水',
	extend_data MEDIUMBLOB NOT NULL,
	UNIQUE KEY auto_id (auto_id) USING BTREE,
	KEY (player_account),
	KEY (player_shortId),
	KEY (settle_time)
	)ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
	`

	// MysqlCheckReportBalanceChangeRecord 用户余额变动报表
	MysqlCheckReportBalanceChangeRecord string = `
	CREATE TABLE IF NOT EXISTS report_balance_change_record(
	auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
	player_uuid varchar(128) NOT NULL,
	player_shortId varchar(10) NOT NULL,
	player_account varchar(128) NOT NULL,
	change_time bigint(20) not null,
	change_reason int(11) not null,
	before_balance bigint(64) not null,
	change_amount bigint(64) not null,
	final_balance bigint(64) not null,
	UNIQUE KEY auto_id (auto_id) USING BTREE,
	KEY (player_account),
	KEY (player_shortId),
	KEY (change_time)
	)ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
	`

	// MysqlCheckReportBonusRecord 奖池报表
	MysqlCheckReportBonusRecord string = `
	CREATE TABLE IF NOT EXISTS report_bonus_change_record(
	auto_id bigint(20) unsigned NOT NULL AUTO_INCREMENT,
	player_uuid varchar(128) NOT NULL,
	player_shortId varchar(10) NOT NULL,
	player_change_balance bigint(20) NOT NULL,

	game_type int(11) NOT NULL,
	game_scene int(11) NOT NULL,
	change_time bigint(20) NOT NULL,

	before_bonus_num bigint(20) NOT NULL,
	change_bonus_num bigint(20) NOT NULL,
	after_bonus_num bigint(20) NOT NULL,
	
	before_system_num bigint(20) NOT NULL,
	change_system_num bigint(20) NOT NULL,
	after_system_num bigint(20) NOT NULL,

	system_ratio int(11) NOT NULL,
	bonus_name varchar(128) NOT NULL,
	change_reason int(11) NOT NULL,

	UNIQUE KEY auto_id (auto_id) USING BTREE,
	KEY (player_uuid),
	KEY (player_shortId),
	KEY (change_time),
	KEY (bonus_name),
	KEY (change_reason)
	)ENGINE=InnoDB 
	AUTO_INCREMENT=1
	DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
	`
)
