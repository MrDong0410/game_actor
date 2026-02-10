# Game Actor

This is the game_actor project.

这个是一个golang项目，用于实现游戏中的actor模型。

主要功能：
1. 支持游戏房管理
2. 每一个游戏房都有属于自己的actor

缺陷
1. 每一个游戏房都是本地缓存，不支持跨节点通信
2. 因为需要网关层面的支持，同一个房间的流量需要路由到同一个节点
