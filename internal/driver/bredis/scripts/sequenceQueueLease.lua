-- Lease transitions use the persisted phase field:
--   phase=ready -> phase=owned on successful acquisition.
--   phase=owned -> phase=owned on successful renewal.
-- phase=corrupt or an unknown phase is isolated and cannot be acquired.

local orderQueue = KEYS[1]
local orderState = KEYS[2]
local scheduler = KEYS[3]
local partitionState = KEYS[4]
local isolation = KEYS[5]

local operation = ARGV[1]
local group = ARGV[2]
local schedulerID = ARGV[3]
local orderKey = ARGV[4]
local consumer = ARGV[5]
local acquisitionID = ARGV[6]
local leaseMS = tonumber(ARGV[7])

local function result(code, value, number)
    return {code, value or '', tostring(number or 0)}
end

local function discard(reason)
    redis.call('XACK', scheduler, group, schedulerID)
    redis.call('XDEL', scheduler, schedulerID)
    redis.call('XADD', isolation, '*',
        'reason', reason, 'scheduler_id', schedulerID,
        'order_key', orderKey, 'consumer', consumer)
end

if operation ~= 'acquire' and operation ~= 'renew' then
    return result('BAD_OPERATION')
end
if leaseMS == nil or leaseMS <= 0 then
    return result('BAD_LEASE')
end
if orderKey == '' or consumer == '' or acquisitionID == '' then
    return result('BAD_IDENTITY')
end
if redis.call('EXISTS', orderState) == 0 then
    if redis.call('LLEN', orderQueue) > 0 then
        discard('orphan_list')
        return result('ISOLATED')
    end
    discard('orphan_token')
    return result('ORPHAN_CLEANED')
end
local stateValues = redis.call('HMGET', orderState,
    'order_key', 'scheduler_id', 'phase', 'depth',
    'acquisition_id', 'consumer', 'deadline_ms')
local stateOrderKey = stateValues[1]
local stateSchedulerID = stateValues[2]
local phase = stateValues[3]
local depth = tonumber(stateValues[4] or '-1')
local currentAcquisition = stateValues[5] or ''
local currentConsumer = stateValues[6] or ''
local currentDeadline = tonumber(stateValues[7] or '0')
if stateOrderKey ~= orderKey then
    discard('order_key_mismatch')
    return result('ISOLATED')
end
if stateSchedulerID ~= schedulerID then
    discard('stale_token')
    return result('STALE_CLEANED')
end
if phase == 'corrupt' or (phase ~= 'ready' and phase ~= 'owned') or depth <= 0 then
    redis.call('HSET', orderState, 'phase', 'corrupt', 'corrupt_reason', 'invalid_state')
    discard('invalid_state')
    return result('ISOLATED')
end

local pending = redis.call('XPENDING', scheduler, group, schedulerID, schedulerID, 1)
if #pending == 0 then
    return result('NOT_PENDING')
end
if pending[1][2] ~= consumer then
    return result('PEL_OWNER_MISMATCH')
end

local head = redis.call('LINDEX', orderQueue, 0)
if not head then
    local count = tonumber(redis.call('HGET', partitionState, 'count') or '0')
    count = math.max(0, count - math.max(0, depth))
    redis.call('HSET', partitionState, 'count', tostring(count))
    discard('empty_queue')
    redis.call('DEL', orderState, orderQueue)
    return result('EMPTY_CLEANED', '', count)
end

local now = redis.call('TIME')
local nowMS = tonumber(now[1]) * 1000 + math.floor(tonumber(now[2]) / 1000)
local deadline = nowMS + leaseMS

if operation == 'renew' then
    if phase ~= 'owned' or currentAcquisition ~= acquisitionID or currentConsumer ~= consumer then
        return result('STALE_ACQUISITION')
    end
    local claimed = redis.call('XCLAIM', scheduler, group, consumer, 0, schedulerID, 'IDLE', 0, 'JUSTID')
    if #claimed == 0 then
        return result('NOT_PENDING')
    end
    redis.call('HSET', orderState, 'deadline_ms', tostring(deadline))
    return result('RENEWED', acquisitionID, deadline)
end

if currentAcquisition ~= '' and currentDeadline > nowMS then
    if currentAcquisition == acquisitionID and currentConsumer == consumer then
        return result('ACQUIRED', head, currentDeadline)
    end
    return result('BUSY', currentAcquisition, currentDeadline)
end

redis.call('HSET', orderState,
    'phase', 'owned',
    'acquisition_id', acquisitionID,
    'consumer', consumer,
    'deadline_ms', tostring(deadline))
return result('ACQUIRED', head, deadline)
