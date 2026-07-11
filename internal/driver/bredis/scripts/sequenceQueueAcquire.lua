local customerQueue = KEYS[1]
local customerState = KEYS[2]
local scheduler = KEYS[3]

local group = ARGV[1]
local schedulerID = ARGV[2]
local customerID = ARGV[3]
local owner = ARGV[4]
local consumer = ARGV[5]
local leaseMS = tonumber(ARGV[6])

if leaseMS == nil or leaseMS <= 0 then
    return {'BAD_LEASE'}
end
if redis.call('EXISTS', customerState) == 0 then
    return {'NO_STATE'}
end
if redis.call('HGET', customerState, 'customer_id') ~= customerID then
    return {'CUSTOMER_MISMATCH'}
end
if redis.call('HGET', customerState, 'scheduler_id') ~= schedulerID then
    return {'STALE_TOKEN'}
end

local pending = redis.call('XPENDING', scheduler, group, schedulerID, schedulerID, 1)
if #pending == 0 then
    return {'NOT_PENDING'}
end
if pending[1][2] ~= consumer then
    return {'PEL_OWNER_MISMATCH'}
end

local now = redis.call('TIME')
local nowMS = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local phase = redis.call('HGET', customerState, 'phase')
if phase == 'owned' then
    local currentDeadline = tonumber(redis.call('HGET', customerState, 'deadline_ms') or '0')
    if currentDeadline > nowMS then
        return {'STATE_BUSY'}
    end
elseif phase ~= 'ready' then
    return {'STATE_BUSY'}
end

local head = redis.call('LINDEX', customerQueue, 0)
if not head then
    return {'EMPTY'}
end

local deadline = nowMS + leaseMS
redis.call('HSET', customerState,
    'phase', 'owned',
    'owner', owner,
    'consumer', consumer,
    'deadline_ms', tostring(deadline))

return {'ACQUIRED', head, tostring(deadline)}
