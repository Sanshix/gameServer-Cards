package common

const (
	// MessageLockAccountLogin 账号登陆锁，细锁，以账号为细锁标记
	MessageLockAccountLogin string = "lock:login:account:"
	// MessageLockAccount 账号操作锁，细锁，以账号为细锁标记
	MessageLockAccount string = "lock:account:"
	// MessageLockAccountRegister 账号注册锁，细锁，以账号为细锁标记
	MessageLockAccountRegister string = "lock:register:account:"
	// MessageLockPlayer 玩家操作锁，细锁，以玩家uuid为细锁标记
	MessageLockPlayer string = "lock:player:"
	// MessageLockMatchingRoom 房间匹配锁，细锁，以游戏类型和游戏场次为细锁标记
	MessageLockMatchingRoom string = "lock:game:match:"
	//MessageLockRobotManagerInfo 机器人管理信息锁，粗锁
	MessageLockRobotManagerInfo string = "lock:robot:manager:info"
	// MessageLockClubInfo 俱乐部信息数据操作锁，细锁，加俱乐部UUID
	MessageLockClubInfo string = "lock:club:info:"
	// MessageLockRoomCodeInfo 房间码信息锁，粗锁
	MessageLockRoomCodeInfo string = "lock:roomCode:info"
	// MessageLockBlood 血池操作锁，粗锁
	MessageLockBlood string = "lock:blood:"
	// MessageLockBonusConfig 血池配置操作锁，粗锁
	MessageLockBonusConfig string = "lock:bonus:config:"
	// MessageLockAgentSettle 代理结算锁，细锁，以代理id为标记
	MessageLockAgentSettle string = "lock:agent:settle:"
	// MessageLockBonus 奖池锁，细锁
	MessageLockBonus string = "lock:bonus:"
	// MessageLockLeaderNabob  富豪排行榜锁，粗锁
	MessageLockLeaderNabob string = "lock:leader:nabob"
	// MessageLockLeaderBigWinner  大赢家排行榜锁，粗锁
	MessageLockLeaderBigWinner string = "lock:leader:bigwinner"
	// MessageLockRobotBonus 机器人奖池锁，细锁,游戏类型及场次
	MessageLockRobotBonus string = "lock:bonus:robot:"
	// MessageBankAccess 银行存取锁，细锁,玩家uuid
	MessageBankAccess string = "lock:bank:"

	// MessageLockLeagueInfo 大联盟信息锁，以联盟ID为标记
	MessageLockLeagueInfo string = "lock:league:"
	// MessageLockLeagueBean 大联盟玩家金豆锁，以联盟id和玩家id为标记
	MessageLockLeagueBean string = "lock:league:bean:"
	// MessageLockLeagueMemberInfo 大联盟玩家锁，(key:联盟id，玩家id)
	//MessageLockLeagueMemberInfo string = "lock:league:member:"
	//MessageLockLeagueCommissionTotal 代理未领取的分成金额（联盟id:玩家id）
	MessageLockLeagueCommissionTotal string = "lock:league:commission:"

	//MessageLockAgentExpendedRoomCard 代理房卡变动锁(key:代理id)
	MessageLockAgentExpendedRoomCard string = "lock:agent:expended:roomcard:"
)
