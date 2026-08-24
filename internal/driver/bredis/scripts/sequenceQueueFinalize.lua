-- Finalize transitions use the persisted phase field:
--   phase=owned -> phase=ready when queued messages remain and a successor token is created.
--   phase=owned -> state key deleted when the finalized message was the last message.
--   phase=ready/owned -> phase=corrupt when order-key, counter, or queue-head invariants fail.

local orderQueue = KEYS[1]
local orderState = KEYS[2]
local scheduler = KEYS[3]
local partitionState = KEYS[4]
local isolation = KEYS[5]

local group = ARGV[1]
local schedulerID = ARGV[2]
local orderKey = ARGV[3]
local consumer = ARGV[4]
local acquisitionID = ARGV[5]
local expectedHead = ARGV[6]

local function result(code, value, number)
    return {code, value or '', tostring(number or 0)}
end

local function discard(reason)
    redis.call('XACK', scheduler, group, schedulerID)
    redis.call('XDEL', scheduler, schedulerID)
    redis.call('XADD', isolation, '*', 'reason', reason,
        'scheduler_id', schedulerID, 'order_key', orderKey,
        'consumer', consumer)
end

if orderKey == '' or consumer == '' or acquisitionID == '' then
    return result('BAD_IDENTITY')
end
if redis.call('EXISTS', orderState) == 0 then
    discard('orphan_token')
    return result('ORPHAN_CLEANED')
end
local stateValues = redis.call('HMGET', orderState,
    'order_key', 'scheduler_id', 'phase',
    'acquisition_id', 'consumer', 'depth')
local stateOrderKey = stateValues[1]
local stateSchedulerID = stateValues[2]
local phase = stateValues[3]
local stateAcquisitionID = stateValues[4]
local stateConsumer = stateValues[5]
local depth = tonumber(stateValues[6] or '-1')
if stateOrderKey ~= orderKey then
    redis.call('HSET', orderState, 'phase', 'corrupt', 'corrupt_reason', 'order_key_mismatch')
    discard('order_key_mismatch')
    return result('ISOLATED')
end
if stateSchedulerID ~= schedulerID then
    discard('stale_token')
    return result('STALE_CLEANED')
end
if phase ~= 'owned' or stateAcquisitionID ~= acquisitionID or
   stateConsumer ~= consumer then
    return result('STALE_ACQUISITION')
end

local pendingEntry = redis.call('XPENDING', scheduler, group, schedulerID, schedulerID, 1)
if #pendingEntry == 0 then
    return result('NOT_PENDING')
end
if pendingEntry[1][2] ~= consumer then
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
if head ~= expectedHead then
    redis.call('HSET', orderState, 'phase', 'corrupt', 'corrupt_reason', 'head_mismatch')
    discard('head_mismatch')
    return result('HEAD_MISMATCH')
end

local count = tonumber(redis.call('HGET', partitionState, 'count') or '-1')
local actualDepth = redis.call('LLEN', orderQueue)
if count <= 0 or depth <= 0 or depth ~= actualDepth then
    redis.call('HSET', orderState, 'phase', 'corrupt', 'corrupt_reason', 'counter_mismatch')
    discard('counter_mismatch')
    return result('ISOLATED')
end

redis.call('LPOP', orderQueue)
count = redis.call('HINCRBY', partitionState, 'count', -1)
depth = redis.call('HINCRBY', orderState, 'depth', -1)
redis.call('XACK', scheduler, group, schedulerID)
redis.call('XDEL', scheduler, schedulerID)

if depth > 0 then
    local successorID = redis.call('XADD', scheduler, '*', 'order_key', orderKey)
    redis.call('HSET', orderState,
        'scheduler_id', successorID,
        'phase', 'ready',
        'acquisition_id', '',
        'consumer', '',
        'deadline_ms', '0')
    return result('SUCCESSOR', successorID, depth)
end

redis.call('DEL', orderState, orderQueue)
return result('EMPTY', '', 0)
