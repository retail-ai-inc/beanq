local key = KEYS[1]
local data = ARGV

for i = 1, #data, 2 do
    local field = data[i]
    local value = data[i + 1]
    redis.call('HSET', key, field, value)
end

local ttl = 3600*6
redis.call('EXPIRE',key,ttl)

return true