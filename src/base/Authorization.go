package base

import (
	"gameServer-demo/src/common"
	pb "gameServer-demo/src/grpc"
	"strconv"
	"strings"
)

//Authorization	鉴权组件
func init() {
	common.AllComponentMap["Authorization"] = &Authorization{}
}

type Authorization struct {
	common.AuthorizationI
	Base
	defaultAuthInfo map[pb.Roles]map[pb.AuthorizationDef]string //默认权限
}

func (c *Authorization) LoadComponent(config *common.OneComponentConfig, componentName string) {
	c.Base.LoadComponent(config, componentName)

	c.defaultAuthInfo = make(map[pb.Roles]map[pb.AuthorizationDef]string)

	c.initDefaultAuth(pb.Roles_Guest, (*c.Config)["guestDefaultAuth"])
	c.initDefaultAuth(pb.Roles_Player, (*c.Config)["playerDefaultAuth"])
	c.initDefaultAuth(pb.Roles_Agent, (*c.Config)["agentDefaultAuth"])
	c.initDefaultAuth(pb.Roles_Manager, (*c.Config)["managerDefaultAuth"])

	common.LogInfo("Authorization component success loaded")

}

//initDefaultAuth 初始化默认权限
func (c *Authorization) initDefaultAuth(role pb.Roles, config string) {
	c.defaultAuthInfo[role] = make(map[pb.AuthorizationDef]string)
	split := strings.Split(config, ",")
	for index := range split {
		if authId, err := strconv.ParseInt(split[index], 10, 32); err == nil {
			if authName, ok := pb.AuthorizationDef_name[int32(authId)]; ok {
				c.defaultAuthInfo[role][pb.AuthorizationDef(authId)] = authName
			}
		}
	}
}

//Verify 验证接口权限
//支持多角色判断 c.Verify("Login_Login",pb.Roles_Agent,[1,2,3])
func (c *Authorization) Verify(authAddr string, userRole pb.Roles, userAuthMap map[pb.AuthorizationDef]pb.AuthorizationDef) bool {
	common.LogDebug("auth verify:", authAddr, userRole, userAuthMap)
	requestedAuthAddr := pb.AuthorizationDef(pb.AuthorizationDef_value[authAddr])
	common.LogDebug("auth check:", requestedAuthAddr)
	//判断是否是guest
	//如果接口是公共接口则直接放行
	if _, ok := c.defaultAuthInfo[pb.Roles_Guest][requestedAuthAddr]; ok {
		return true
	}

	//判断当前角色默认权限能否访问
	if _, ok := c.defaultAuthInfo[userRole][requestedAuthAddr]; !ok {
		return false
	}

	//如果用户权限列表为nil，则表示使用默认权限配置
	if nil == userAuthMap || len(userAuthMap) == 0 {
		return true
	}
	//判断用户是否包含权限
	if _, ok := userAuthMap[requestedAuthAddr]; ok {
		return true
	}
	return false

}

func (c *Authorization) GetDefaultAuth(role pb.Roles) map[pb.AuthorizationDef]string {
	if data, ok := c.defaultAuthInfo[role]; ok {
		return data
	}
	return nil
}
