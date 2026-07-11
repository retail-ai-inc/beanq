local customerQueue = KEYS[1]
local customerState = KEYS[2]
local scheduler = KEYS[3]
local pending = KEYS[4]

local group = ARGV[1]
local schedulerID = ARGV[2]
local customerID = ARGV[3]
local owner = ARGV[4]
local consumer = ARGV[5]
local expectedHead = ARGV[6]

if redis.call('EXISTS', customerState) == 0 then
    return {'NO_STATE'}
end
if redis.call('HGET', customerState, 'customer_id') ~= customerID then
    return {'CUSTOMER_MISMATCH'}
end
if redis.call('HGET', customerState, 'scheduler_id') ~= schedulerID then
    return {'STALE_TOKEN'}
end
if redis.call('HGET', customerState, 'phase') ~= 'owned' or
   redis.call('HGET', customerState, 'owner') ~= owner or
   redis.call('HGET', customerState, 'consumer') ~= consumer then
    return {'STALE_OWNER'}
end

local pel = redis.call('XPENDING', scheduler, group, schedulerID, schedulerID, 1)
if #pel == 0 then
    return {'NOT_PENDING'}
end
if pel[1][2] ~= consumer then
    return {'PEL_OWNER_MISMATCH'}
end

local head = redis.call('LINDEX', customerQueue, 0)
if not head then
    return {'EMPTY'}
end
if head ~= expectedHead then
    return {'HEAD_MISMATCH'}
end

redis.call('LPOP', customerQueue)
local count = tonumber(redis.call('GET', pending) or '0')
if count > 1 then
    redis.call('DECR', pending)
else
    redis.call('SET', pending, 0)
end
redis.call('XACK', scheduler, group, schedulerID)
redis.call('XDEL', scheduler, schedulerID)

local remaining = redis.call('LLEN', customerQueue)
if remaining > 0 then
    local successorID = redis.call('XADD', scheduler, '*', 'customer_id', customerID)
    redis.call('HSET', customerState,
        'scheduler_id', successorID,
        'phase', 'ready',
        'owner', '',
        'consumer', '',
        'deadline_ms', '0')
    return {'SUCCESSOR', successorID, tostring(remaining)}
end

redis.call('DEL', customerState)
redis.call('DEL', customerQueue)
return {'EMPTY', '0'}
