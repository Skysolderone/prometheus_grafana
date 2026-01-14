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
