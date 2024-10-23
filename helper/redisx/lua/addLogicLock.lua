local key = KEYS[1]
local val = redis.call('GET',key)
if val == '1' then
    return 1
end

redis.call('SETEX',key,60,'1')
return 0