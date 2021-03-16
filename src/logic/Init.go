package logic

import (
	Hall "gameServer-demo/src/logic/Hall"
	PushBobbin "gameServer-demo/src/logic/PushBobbin"
	Robot "gameServer-demo/src/logic/Robot"
)

// Init 用于方便包被外部引用的函数，同时在这里引用子包
func Init() {
	PushBobbin.Init()
	Hall.Init()
	Robot.Init()
}
