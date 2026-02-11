package room

import (
	"game_actor/match"
	"sync"
	"sync/atomic"

	"github.com/samber/lo"
)

type BaseRoom struct {
	Status    atomic.Int32
	RoomID    int64
	matchInfo *match.MatchInfo
	players   sync.Map
	playerNum atomic.Int32
}

func NewBaseRoom(roomID int64, matchInfo *match.MatchInfo) *BaseRoom {
	baseRoom := new(BaseRoom)
	baseRoom.RoomID = roomID
	baseRoom.matchInfo = matchInfo
	return baseRoom
}

func (r *BaseRoom) GetRoomID() int64 {
	return r.RoomID
}

func (r *BaseRoom) UserEnterRoom(uid int64, roomID int64) {
	// 检查用户是否在匹配列表中
	// 如果不是，直接返回
	if !lo.ContainsBy(r.matchInfo.Players, func(player *match.Player) bool {
		return player.PlayerUID == uid
	}) {
		return
	}

	// 尝试将用户加入房间
	// LoadOrStore: 如果键存在，加载并返回 true；如果不存在，存储并返回 false
	// 如果返回 true (loaded)，说明用户已经在房间里了，直接返回
	if _, loaded := r.players.LoadOrStore(uid, true); loaded {
		return
	}

	// 用户成功加入，增加人数
	r.playerNum.Add(1)
}

func (r *BaseRoom) UserLeaveRoom(uid int64, roomID int64) {
	// 实现离开房间的逻辑
	if _, loaded := r.players.LoadAndDelete(uid); loaded {
		r.playerNum.Add(-1)
	}
}

func (r *BaseRoom) GetPlayers() []int64 {
	var uids []int64
	r.players.Range(func(key, value any) bool {
		if uid, ok := key.(int64); ok {
			uids = append(uids, uid)
		}
		return true
	})
	return uids
}
