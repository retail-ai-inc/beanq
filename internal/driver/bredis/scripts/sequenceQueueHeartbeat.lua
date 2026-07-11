local customerState = KEYS[1]
local scheduler = KEYS[2]

local group = ARGV[1]
local schedulerID = ARGV[2]
local owner = ARGV[3]
local consumer = ARGV[4]
local leaseMS = tonumber(ARGV[5])

if leaseMS == nil or leaseMS <= 0 then
    return {'BAD_LEASE'}
end
if redis.call('EXISTS', customerState) == 0 then
    return {'NO_STATE'}
end
if redis.call('HGET', customerState, 'scheduler_id') ~= schedulerID then
    return {'STALE_TOKEN'}
end
if redis.call('HGET', customerState, 'phase') ~= 'owned' or
   redis.call('HGET', customerState, 'owner') ~= owner or
   redis.call('HGET', customerState, 'consumer') ~= consumer then
    return {'STALE_OWNER'}
end

local pending = redis.call('XPENDING', scheduler, group, schedulerID, schedulerID, 1)
if #pending == 0 then
    return {'NOT_PENDING'}
end
if pending[1][2] ~= consumer then
    return {'PEL_OWNER_MISMATCH'}
end

local claimed = redis.call('XCLAIM', scheduler, group, consumer, 0, schedulerID, 'IDLE', 0, 'JUSTID')
if #claimed == 0 then
    return {'NOT_PENDING'}
end

local now = redis.call('TIME')
local nowMS = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local deadline = nowMS + leaseMS
redis.call('HSET', customerState, 'deadline_ms', tostring(deadline))
return {'HEARTBEAT', tostring(deadline)}
