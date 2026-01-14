#!/bin/bash

# 目标地址（按需修改）
BASE_URL="http://localhost:8080"

# 线程数 / 并发数 / 时长
THREADS=4
CONNECTIONS=100
DURATION="60s"

# 高并发测试参数（用于测试并发效果）
HIGH_CONCURRENCY=500
HIGH_DURATION="30s"

# 生成 post.lua
cat > post.lua << 'POST_EOF'
wrk.method = "POST"
wrk.body   = '{"name":"test"}'
wrk.headers["Content-Type"] = "application/json"
POST_EOF

# 生成 mix.lua：随机压 /get 和 /post
cat > mix.lua << 'MIX_EOF'
paths = { "/get", "/post" }

request = function()
  local path = paths[ math.random(1, #paths) ]

  if path == "/post" then
    wrk.method = "POST"
    wrk.body   = '{"name":"test"}'
    wrk.headers["Content-Type"] = "application/json"
  else
    wrk.method = "GET"
    wrk.body   = nil
  end

  return wrk.format(nil, path)
end
MIX_EOF

echo "=== 压测 /get ==="
wrk -t${THREADS} -c${CONNECTIONS} -d${DURATION} ${BASE_URL}/get

echo

echo "=== 压测 /post（POST JSON）==="
wrk -t${THREADS} -c${CONNECTIONS} -d${DURATION} -s post.lua ${BASE_URL}/post

echo

echo "=== 混合压测 /get + /post ==="
wrk -t${THREADS} -c${CONNECTIONS} -d${DURATION} -s mix.lua ${BASE_URL}

echo

echo "=== 高并发压测 /slow（可以看到明显的并发数）==="
echo "提示：在另一个终端运行 'watch -n 1 \"curl -s http://localhost:8080/metrics | grep http_requests_in_flight\"' 观察并发数"
wrk -t${THREADS} -c${HIGH_CONCURRENCY} -d${HIGH_DURATION} ${BASE_URL}/slow
