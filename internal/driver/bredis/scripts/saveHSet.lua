local key = KEYS[1]

if #ARGV > 0 then
    redis.call('HSET', key, unpack(ARGV))
end

local ttl = 3600*6
redis.call('EXPIRE',key,ttl)

return true