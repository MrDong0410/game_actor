package room

import (
	"fmt"
	"game_actor/match"
	"game_actor/session"
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
	channels  sync.Map // channelID (string) -> *Channel
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

// 检查房间是否有玩家
func (r *BaseRoom) Check() bool {
	return r.Status.Load() == RoomStatus_Init && r.playerNum.Load() > 0
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

func (r *BaseRoom) UserEnterRoom(uid int64, roomID int64, sess session.Session) {
	// 绑定 Session 到默认频道（RoomID）
	if sess != nil {
		channelID := fmt.Sprintf("%d", roomID)
		r.JoinChannel(channelID, uid, sess)
	}

	// 玩家已经进入了
	if _, loaded := r.players.LoadOrStore(uid, true); loaded {
		return
	}
	// 只有初始化状态才能加入(玩家)，初始阶段不能进入观众？
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

func (r *BaseRoom) KickUser(uid int64) {
	// 从默认频道获取 Session 并关闭
	channelID := fmt.Sprintf("%d", r.RoomID)
	if val, ok := r.channels.Load(channelID); ok {
		channel := val.(*Channel)
		if sess, ok := channel.GetSession(uid); ok {
			sess.Close()
		}
	}
	r.UserLeaveRoom(uid, r.RoomID)
}

func (r *BaseRoom) UserLeaveRoom(uid int64, roomID int64) {
	// 移除 Session (默认从 RoomID 频道移除)
	channelID := fmt.Sprintf("%d", roomID)
	r.LeaveChannel(channelID, uid)

	// 判断是否是玩家
	isPlayer := lo.ContainsBy(r.matchInfo.Players, func(player *match.Player) bool {
		return player.PlayerUID == uid
	})

	// 如果是玩家，需要从 map 中移除并减少人数
	if isPlayer {
		if _, loaded := r.players.LoadAndDelete(uid); loaded {
			r.playerNum.Add(-1)
		}
	}

	// 玩家离开了，这里需要通知游戏房玩家离开了
	for _, opt := range r.option.playerOpts {
		opt.OnLeave(uid, isPlayer)
	}
}

func (r *BaseRoom) JoinChannel(channelID string, uid int64, sess session.Session) {
	val, _ := r.channels.LoadOrStore(channelID, NewChannel(channelID))
	channel := val.(*Channel)
	channel.Add(uid, sess)
}

func (r *BaseRoom) LeaveChannel(channelID string, uid int64) {
	if val, ok := r.channels.Load(channelID); ok {
		channel := val.(*Channel)
		channel.Remove(uid)
	}
}

func (r *BaseRoom) Broadcast(channelID string, msg []byte) {
	if val, ok := r.channels.Load(channelID); ok {
		channel := val.(*Channel)
		channel.Broadcast(msg)
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
