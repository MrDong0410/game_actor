package room

import (
	"game_actor/session"
	"sync"
)

type Channel struct {
	ID       string
	sessions sync.Map // uid -> session.Session
}

func NewChannel(id string) *Channel {
	return &Channel{
		ID: id,
	}
}

func (c *Channel) Add(uid int64, sess session.Session) {
	c.sessions.Store(uid, sess)
}

func (c *Channel) GetSession(uid int64) (session.Session, bool) {
	if val, ok := c.sessions.Load(uid); ok {
		return val.(session.Session), true
	}
	return nil, false
}

func (c *Channel) Remove(uid int64) {
	c.sessions.Delete(uid)
}

func (c *Channel) Broadcast(msg []byte) {
	c.sessions.Range(func(key, value any) bool {
		sess, ok := value.(session.Session)
		if ok {
			sess.Send(msg)
		}
		return true
	})
}
