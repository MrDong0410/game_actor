package service

import (
	"errors"
	"fmt"
	"game_actor/match"
	"game_actor/room"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

type Builder func(matchInfo *match.MatchInfo) room.GameRoom

type RoomService struct {
	Rooms sync.Map

	Builder   Builder
	scheduler *gocron.Scheduler
}

func NewRoomService(builder Builder) *RoomService {
	s := gocron.NewScheduler(time.UTC)
	s.StartAsync()
	return &RoomService{
		Builder:   builder,
		scheduler: s,
	}
}

func (s *RoomService) CreateRoom(roomID int64, matchInfo *match.MatchInfo) (room.GameRoom, error) {
	_, ok := s.Rooms.Load(roomID)
	if ok {
		return nil, errors.New("room already exist")
	}
	gameRoom := s.Builder(matchInfo)
	_, ok = s.Rooms.LoadOrStore(roomID, gameRoom)
	if ok {
		return nil, errors.New("room already exist")
	}

	// 1. 创建房间之后，根据matchInfo里面的最长等待playMaxWait，判断是否要开始游戏
	if matchInfo.MaxPlayerWaitTime > 0 {
		// 使用 gocron 调度自动开始任务
		// Tag: room-{id}, room-{id}-start
		s.scheduler.Every(matchInfo.MaxPlayerWaitTime).Seconds().LimitRunsTo(1).Tag(
			fmt.Sprintf("room-%d", roomID),
			fmt.Sprintf("room-%d-start", roomID),
		).Do(s.StartRoom, roomID)
	}

	return gameRoom, nil
}

func (s *RoomService) GetRoom(roomID int64) (room.GameRoom, bool) {
	gameRoom, ok := s.Rooms.Load(roomID)
	if !ok {
		return nil, false
	}
	return gameRoom.(room.GameRoom), true
}

func (s *RoomService) StartRoom(roomID int64) error {
	gameRoom, ok := s.GetRoom(roomID)
	if !ok {
		return errors.New("room not exist")
	}

	// 取消自动开始任务（如果是手动开始的，防止重复触发）
	s.scheduler.RemoveByTag(fmt.Sprintf("room-%d-start", roomID))
	// 判断是否需要开始游戏
	if !gameRoom.Check() {
		return errors.New("room not ready")
	}
	// 调用房间的 Start 方法
	// 注意：Start 内部应该处理并发调用，确保只能启动一次
	gameRoom.Start()
	// 获取匹配信息
	matchInfo := gameRoom.GetMatchInfo()
	// 2. 游戏开始之后，要根据游戏最长时间，要自动关闭游戏
	if matchInfo != nil && matchInfo.MaxGameTime > 0 {
		// 使用 gocron 调度自动关闭任务
		// Tag: room-{id}, room-{id}-close
		s.scheduler.Every(matchInfo.MaxGameTime).Seconds().LimitRunsTo(1).Tag(
			fmt.Sprintf("room-%d", roomID),
			fmt.Sprintf("room-%d-close", roomID),
		).Do(s.CloseRoom, roomID)
	}
	return nil
}

func (s *RoomService) CloseRoom(roomID int64) error {
	gameRoom, ok := s.Rooms.Load(roomID)
	if !ok {
		return errors.New("room not exist")
	}
	// 取消该房间的所有调度任务
	s.scheduler.RemoveByTag(fmt.Sprintf("room-%d", roomID))
	s.Rooms.Delete(roomID)
	// 关闭房间
	gameRoom.(room.GameRoom).Close()
	return nil
}

// Keep DeleteRoom for backward compatibility or alias to CloseRoom
func (s *RoomService) DeleteRoom(roomID int64) error {
	return s.CloseRoom(roomID)
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
