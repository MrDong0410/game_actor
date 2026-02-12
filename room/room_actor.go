package room

import (
	"game_actor/match"
	"game_actor/session"

	"github.com/vladopajic/go-actor/actor"
)

type RoomActor struct {
	*BaseRoom
	actor   actor.Actor
	mailbox actor.MailboxSender[func()]
}

type roomWorker struct {
	mailbox actor.MailboxReceiver[func()]
}

func (w *roomWorker) DoWork(ctx actor.Context) actor.WorkerStatus {
	select {
	case <-ctx.Done():
		return actor.WorkerEnd
	case fn := <-w.mailbox.ReceiveC():
		fn()
		return actor.WorkerContinue
	}
}

func NewRoomActor(roomID int64, matchInfo *match.MatchInfo, opts ...OptionFunc) *RoomActor {
	mbx := actor.NewMailbox[func()]()
	worker := &roomWorker{mailbox: mbx}
	a := actor.New(worker)
	a.Start()

	return &RoomActor{
		BaseRoom: NewBaseRoom(roomID, matchInfo, opts...),
		actor:    a,
		mailbox:  mbx,
	}
}

// Invoke 异步投递任务，不等待结果
func (r *RoomActor) Invoke(f func()) error {
	return r.mailbox.Send(nil, f)
}

// SyncInvoke 同步投递任务，等待执行结果
func (r *RoomActor) SyncInvoke(f func() (any, error)) (any, error) {
	resultChan := make(chan any, 1)
	errChan := make(chan error, 1)
	err := r.mailbox.Send(nil, func() {
		res, fnErr := f()
		resultChan <- res
		errChan <- fnErr
	})
	if err != nil {
		return nil, err
	}
	return <-resultChan, <-errChan
}

func (r *RoomActor) UserEnterRoom(uid int64, roomID int64, sess session.Session) {
	r.Invoke(func() {
		r.BaseRoom.UserEnterRoom(uid, roomID, sess)
	})
}

func (r *RoomActor) UserLeaveRoom(uid int64, roomID int64) {
	r.Invoke(func() {
		r.BaseRoom.UserLeaveRoom(uid, roomID)
	})
}

func (r *RoomActor) KickUser(uid int64) {
	r.Invoke(func() {
		r.BaseRoom.KickUser(uid)
	})
}

func (r *RoomActor) Start() {
	r.Invoke(func() {
		r.BaseRoom.Start()
	})
}

func (r *RoomActor) Close() {
	// 使用 SyncInvoke 等待关闭逻辑完成
	r.SyncInvoke(func() (any, error) {
		r.BaseRoom.Close()
		return nil, nil
	})

	// 停止 actor
	r.actor.Stop()
}

func (r *RoomActor) Broadcast(channelID string, msg []byte) {
	r.Invoke(func() {
		r.BaseRoom.Broadcast(channelID, msg)
	})
}

func (r *RoomActor) JoinChannel(channelID string, uid int64, sess session.Session) {
	r.Invoke(func() {
		r.BaseRoom.JoinChannel(channelID, uid, sess)
	})
}

func (r *RoomActor) LeaveChannel(channelID string, uid int64) {
	r.Invoke(func() {
		r.BaseRoom.LeaveChannel(channelID, uid)
	})
}
