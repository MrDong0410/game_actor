package service

import (
	"errors"
	"game_actor/room"
	"sync"
)

type Builder func() room.GameRoom

type RoomService struct {
	Rooms sync.Map

	Builder Builder
}

func NewRoomService() *RoomService {
	return &RoomService{}
}

func (s *RoomService) CreateRoom(roomID int64) error {
	_, ok := s.Rooms.Load(roomID)
	if ok {
		return errors.New("room already exist")
	}
	gameRoom := s.Builder()
	s.Rooms.Store(roomID, gameRoom)
	return nil
}

func (s *RoomService) GetRoom(roomID int64) (room.GameRoom, bool) {
	gameRoom, ok := s.Rooms.Load(roomID)
	if !ok {
		return nil, false
	}
	return gameRoom.(room.GameRoom), true
}
