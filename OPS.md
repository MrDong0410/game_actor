# 运维配置指南 (OPS Guide)

本文档旨在指导运维人员配置 Nginx/OpenResty，实现游戏服务器的动态路由和负载均衡。

## 1. 架构概述

我们使用 OpenResty (Nginx + Lua) 作为网关，结合 Redis 实现动态路由。

*   **Redis Key**: `game:nodes:{node_id}` -> `host:port`
*   **路由规则**:
    *   如果 URL 参数或 Header 中携带 `node=node-1`，则路由到 `node-1`。
    *   如果没有携带 `node` 参数，则随机或轮询路由到任意可用节点（本示例简化为返回错误或默认节点，实际可结合 Redis `KEYS game:nodes:*` 做负载均衡）。

## 2. 前置要求

*   OpenResty (推荐 1.19.3.1+)
*   `lua-resty-redis` 库 (OpenResty 自带)
*   Redis 服务

## 3. Nginx 配置示例

将以下配置保存为 `nginx.conf` 或在 `conf.d/game.conf` 中引用。

```nginx
worker_processes  1;
error_log logs/error.log;

events {
    worker_connections 1024;
}

http {
    # 配置 Redis 地址
    upstream redis_backend {
        server 127.0.0.1:6379;
        keepalive 1024;
    }

    # 定义 Lua 共享内存（可选，用于缓存）
    lua_shared_dict node_cache 10m;

    server {
        listen 80;
        server_name game.example.com;

        # 1. WebSocket 路由
        location /ws {
            # 必须设置，允许 WebSocket 升级
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            
            # 设置变量用于 Lua 赋值
            set $target_node "";
            
            access_by_lua_block {
                local redis = require "resty.redis"
                local red = redis:new()
                
                red:set_timeout(1000) -- 1 sec
                
                -- 连接 Redis
                local ok, err = red:connect("127.0.0.1", 6379)
                if not ok then
                    ngx.log(ngx.ERR, "failed to connect to redis: ", err)
                    return ngx.exit(500)
                end
                
                -- 获取 node 参数 (Query Param > Header)
                local args = ngx.req.get_uri_args()
                local node_id = args["node"]
                if not node_id then
                    node_id = ngx.req.get_headers()["x-node-id"]
                end
                
                if not node_id then
                    -- 策略 A: 报错，必须指定节点
                    -- ngx.status = 400
                    -- ngx.say("Missing node parameter")
                    -- return ngx.exit(400)
                    
                    -- 策略 B: 随机获取一个节点 (简单示例，生产环境建议维护一个 Set)
                    -- local keys, err = red:keys("game:nodes:*")
                    -- if not keys or #keys == 0 then
                    --     ngx.status = 503
                    --     ngx.say("No available nodes")
                    --     return ngx.exit(503)
                    -- end
                    -- local random_key = keys[math.random(#keys)]
                    -- node_id = string.sub(random_key, 12) -- remove "game:nodes:" prefix
                    
                    -- 策略 C: 默认路由到 node-1 (仅测试用)
                    node_id = "node-1"
                end
                
                -- 从 Redis 获取节点地址
                local key = "game:nodes:" .. node_id
                local host, err = red:get(key)
                
                if not host or host == ngx.null then
                    ngx.status = 404
                    ngx.say("Node not found: " .. node_id)
                    return ngx.exit(404)
                end
                
                -- 将连接放回连接池
                local ok, err = red:set_keepalive(10000, 100)
                
                -- 设置上游地址
                ngx.var.target_node = host
            }
            
            # 代理到目标节点
            proxy_pass http://$target_node;
        }
    }
}
```

## 4. 验证路由

1.  **启动游戏节点**:
    ```bash
    # 终端 1
    go run cmd/main.go --port 8081 --node node-1
    # 终端 2
    go run cmd/main.go --port 8082 --node node-2
    ```

2.  **检查 Redis**:
    ```bash
    redis-cli get game:nodes:node-1
    # 输出: 127.0.0.1:8081
    redis-cli get game:nodes:node-2
    # 输出: 127.0.0.1:8082
    ```

3.  **客户端连接**:
    *   连接 Node 1: `ws://game.example.com/ws?node=node-1`
    *   连接 Node 2: `ws://game.example.com/ws?node=node-2`

## 5. 常见问题

*   **Redis 连接失败**: 检查 Redis 地址配置。
*   **Node not found**: 检查游戏节点是否成功启动并注册到 Redis（Key 是否存在）。
*   **WebSocket 断开**: 检查 `proxy_read_timeout` 和 `proxy_send_timeout` 配置，确保长连接不超时。
