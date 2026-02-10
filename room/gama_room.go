package room

type GameRoom interface {
	// 获取房间ID
	GetRoomID() int64
	// 用户进入房间
	UserEnterRoom(uid int64, roomID int64)
	// 用户离开房间
	UserLeaveRoom(uid int64, roomID int64)
}
