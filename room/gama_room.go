package room

import "game_actor/match"

type GameRoom interface {
	// 获取房间ID
	GetRoomID() int64
	// 用户进入房间
	UserEnterRoom(uid int64, roomID int64)
	// 用户离开房间
	UserLeaveRoom(uid int64, roomID int64)
	// 获取匹配信息
	GetMatchInfo() *match.MatchInfo
	// 检查房间
	Check() bool
	// 开始游戏
	Start()
	// 结束游戏
	Close()
}
