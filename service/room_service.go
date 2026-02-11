package service

import (
	"errors"
	"game_actor/match"
	"game_actor/room"
	"sync"
)

type Builder func(matchInfo *match.MatchInfo) room.GameRoom

type RoomService struct {
	Rooms sync.Map

	Builder Builder
}

func NewRoomService() *RoomService {
	return &RoomService{}
}

func (s *RoomService) CreateRoom(roomID int64, matchInfo *match.MatchInfo) error {
	_, ok := s.Rooms.Load(roomID)
	if ok {
		return errors.New("room already exist")
	}
	gameRoom := s.Builder(matchInfo)
	_, ok = s.Rooms.LoadOrStore(roomID, gameRoom)
	if ok {
		return errors.New("room already exist")
	}
	return nil
}

func (s *RoomService) GetRoom(roomID int64) (room.GameRoom, bool) {
	gameRoom, ok := s.Rooms.Load(roomID)
	if !ok {
		return nil, false
	}
	return gameRoom.(room.GameRoom), true
}

func (s *RoomService) DeleteRoom(roomID int64) error {
	_, ok := s.Rooms.Load(roomID)
	if !ok {
		return errors.New("room not exist")
	}
	s.Rooms.Delete(roomID)
	return nil
}

func (s *RoomService) UserEnterRoom(uid int64, roomID int64) error {
	gameRoom, ok := s.GetRoom(roomID)
	if !ok {
		return errors.New("room not exist")
	}
	gameRoom.UserEnterRoom(uid, roomID)
	return nil
}

func (s *RoomService) UserLeaveRoom(uid int64, roomID int64) error {
	gameRoom, ok := s.GetRoom(roomID)
	if !ok {
		return errors.New("room not exist")
	}
	gameRoom.UserLeaveRoom(uid, roomID)
	return nil
}
