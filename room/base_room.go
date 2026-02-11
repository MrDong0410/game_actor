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
	// 检查用户是否在匹配列表中
	isPlayer := lo.ContainsBy(r.matchInfo.Players, func(player *match.Player) bool {
		return player.PlayerUID == uid
	})

	if isPlayer {
		// 尝试将用户加入房间
		// LoadOrStore: 如果键存在，加载并返回 true；如果不存在，存储并返回 false
		// 如果返回 true (loaded)，说明用户已经在房间里了，直接返回
		if _, loaded := r.players.LoadOrStore(uid, true); loaded {
			return
		}
		// 用户成功加入
		r.playerNum.Add(1)
	}

	// 玩家进入了，这里需要通知游戏房玩家进入了
	for _, opt := range r.option.playerOpts {
		opt.OnEnter(uid, isPlayer)
	}
}

func (r *BaseRoom) UserLeaveRoom(uid int64, roomID int64) {
	// 实现离开房间的逻辑
	// 如果用户不在房间里，直接返回
	val, loaded := r.players.LoadAndDelete(uid)
	if !loaded {
		return
	}

	// 获取用户身份
	isPlayer, ok := val.(bool)
	if !ok {
		// 如果 Map 中存储的不是 bool，兜底重新计算
		isPlayer = lo.ContainsBy(r.matchInfo.Players, func(player *match.Player) bool {
			return player.PlayerUID == uid
		})
	}

	// 只有玩家离开才减少人数
	if isPlayer {
		r.playerNum.Add(-1)
	}

	// 玩家离开了，这里需要通知游戏房玩家离开了
	for _, opt := range r.option.playerOpts {
		opt.OnLeave(uid, isPlayer)
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
