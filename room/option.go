package room

type RoomOption interface {
	OnStart(roomID int64)
	OnClose(roomID int64)
}

type PlayerOption interface {
	OnEnter(uid int64, isPlayer bool)
	OnLeave(uid int64, isPlayer bool)
}

type Option struct {
	roomOpts   []RoomOption
	playerOpts []PlayerOption
}

type OptionFunc func(*Option)

func WithRoomOption(opt RoomOption) OptionFunc {
	return func(o *Option) {
		o.roomOpts = append(o.roomOpts, opt)
	}
}

func WithPlayerOption(opt PlayerOption) OptionFunc {
	return func(o *Option) {
		o.playerOpts = append(o.playerOpts, opt)
	}
}
