package room

import (
	"game_actor/match"
	"game_actor/session"
)

type GameRoom interface {
	// 获取房间ID
	GetRoomID() int64
	// 用户进入房间
	UserEnterRoom(uid int64, roomID int64, sess session.Session)
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
	// 广播消息
	Broadcast(channelID string, msg []byte)
	// 加入频道
	JoinChannel(channelID string, uid int64, sess session.Session)
	// 离开频道
	LeaveChannel(channelID string, uid int64)
	// 剔除用户（关闭Session）
	KickUser(uid int64)
}
