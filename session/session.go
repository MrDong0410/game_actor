package session

type Session interface {
	ID() string
	UserID() int64
	SetUserID(uid int64)
	Send(msg []byte) error
	Close() error
}
