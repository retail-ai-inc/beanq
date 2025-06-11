local streamKey = KEYS[1]
local orderRediKey = KEYS[2]
local expireTime = tonumber(KEYS[3])

local fields = ARGV

local rediKeyStatus = redis.call('HGET',orderRediKey,"status")
if rediKeyStatus == 'pending' then
    return  {err = 'Locking'}
end

local message = {}
for i = 1, #fields, 2 do
    table.insert(message, fields[i])
    table.insert(message, fields[i+1])
end

redis.call('XADD', streamKey, '*', unpack(message))
table.insert(message, 'status')
table.insert(message, 'pending')
redis.call('HSET',orderRediKey,unpack(message))

if expireTime > 0 then
    redis.call('EXPIRE',orderRediKey,expireTime)
end

return true