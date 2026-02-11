package match

/*
	这个目录存放匹配的信息，但其实match info的实际数据应该是pb定义或者json定义
	因为真实游戏一定会有一个匹配服

	因此这个目录下面的所有文件都应该是第三方文件
**/

type MatchInfo struct {
	// 游戏id
	GameID int64
	// 匹配id
	MatchID int64
	// 本地游戏的所有玩家信息
	Players []*Player
}

// 游戏玩家
type Player struct {
	// 玩家id
	PlayerUID int64
	// 阵营
	Camp int32
}
