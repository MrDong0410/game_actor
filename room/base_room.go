package room

import (
	"game_actor/match"
	"sync"
	"sync/atomic"

	"github.com/samber/lo"
)

const (
	RoomStatus_Init  = 0
	RoomStatus_Start = 1
	RoomStatus_Close = 2
)

type BaseRoom struct {
	Status    atomic.Int32
	RoomID    int64
	matchInfo *match.MatchInfo
	players   sync.Map
	playerNum atomic.Int32

	option *Option
}

func NewBaseRoom(roomID int64, matchInfo *match.MatchInfo, opts ...OptionFunc) *BaseRoom {
	baseRoom := new(BaseRoom)
	opt := new(Option)
	for _, optFunc := range opts {
		optFunc(opt)
	}
	baseRoom.RoomID = roomID
	baseRoom.matchInfo = matchInfo
	baseRoom.Status.Store(RoomStatus_Init)
	baseRoom.option = opt
	return baseRoom
}

func (r *BaseRoom) GetRoomID() int64 {
	return r.RoomID
}

func (r *BaseRoom) GetMatchInfo() *match.MatchInfo {
	return r.matchInfo
}

// 检查房间是否完整
func (r *BaseRoom) Check() bool {
	return r.Status.Load() == RoomStatus_Init && r.playerNum.Load() == int32(len(r.matchInfo.Players))
}

func (r *BaseRoom) Start() bool {
	if !r.Status.CompareAndSwap(RoomStatus_Init, RoomStatus_Start) {
		return false
	}
	// 游戏开始,这里需要通知游戏房游戏开始了
	for _, opt := range r.option.roomOpts {
		opt.OnStart(r.RoomID)
	}
	return true
}

func (r *BaseRoom) Close() {
	if !r.Status.CompareAndSwap(RoomStatus_Start, RoomStatus_Close) {
		return
	}
	// 游戏结束，这里需要通知游戏结束了
	for _, opt := range r.option.roomOpts {
		opt.OnClose(r.RoomID)
	}
}

func (r *BaseRoom) UserEnterRoom(uid int64, roomID int64) {
	// 玩家已经进入了
	if _, loaded := r.players.LoadOrStore(uid, true); loaded {
		return
	}
	// 只有初始化状态才能加入(玩家)
	if r.Status.Load() != RoomStatus_Init {
		isPlayer := lo.ContainsBy(r.matchInfo.Players, func(player *match.Player) bool {
			return player.PlayerUID == uid
		})
		// 不是玩家，直接返回
		if !isPlayer {
			return
		}
		// 游戏玩家
		_, loaded := r.players.LoadOrStore(uid, true)
		if loaded {
			return
		}
		r.playerNum.Add(1)
		r.playerEnter(uid)
		return
	}
	// 玩家进入了，这里需要通知游戏房玩家进入了
	for _, opt := range r.option.playerOpts {
		opt.OnEnter(uid, false)
	}
}

func (r *BaseRoom) UserLeaveRoom(uid int64, roomID int64) {
	// 实现离开房间的逻辑
	if _, loaded := r.players.LoadAndDelete(uid); loaded {
		r.playerNum.Add(-1)

		isPlayer := lo.ContainsBy(r.matchInfo.Players, func(player *match.Player) bool {
			return player.PlayerUID == uid
		})

		// 玩家离开了，这里需要通知游戏房玩家离开了
		for _, opt := range r.option.playerOpts {
			opt.OnLeave(uid, isPlayer)
		}
	}
}

func (r *BaseRoom) playerEnter(uid int64) {
	// 玩家进入了，这里需要通知游戏房玩家进入了
	for _, opt := range r.option.playerOpts {
		opt.OnEnter(uid, true)
	}
	if r.playerNum.Load() == int32(len(r.matchInfo.Players)) {
		// 所有玩家都进入了，开始游戏
		r.Start()
	}
}
