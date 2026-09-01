-- The persisted phase field has exactly three values: ready, owned, and corrupt.
-- Order-key state lifecycle:
--   state key absent -> phase=ready: the first message creates state and a scheduler token.
--   phase=ready -> phase=owned: a consumer acquires the queue head.
--   phase=owned -> phase=ready: finalize creates a successor token when messages remain.
--   phase=owned -> state key deleted: finalize completes the last message.
--   phase=ready/owned -> phase=corrupt: an invariant mismatch isolates the order queue.

local orderQueue = KEYS[1]
local orderState = KEYS[2]
local scheduler = KEYS[3]
local partitionState = KEYS[4]
local isolation = KEYS[5]

local payload = ARGV[1]
local orderKey = ARGV[2]
local capacity = tonumber(ARGV[3])

local function result(code, value, number)
    return {code, value or '', tostring(number or 0)}
end

if capacity == nil or capacity <= 0 then
    return result('BAD_CAPACITY')
end

local partitionValues = redis.call('HMGET', partitionState, 'capacity', 'count')
local storedCapacity = tonumber(partitionValues[1] or '0')
local count = tonumber(partitionValues[2] or '0')
if storedCapacity == 0 then
    redis.call('HSET', partitionState, 'capacity', tostring(capacity), 'count', tostring(count))
elseif storedCapacity ~= capacity then
    return result('CONFIG_MISMATCH', tostring(storedCapacity), 0)
end

if count >= capacity then
    return result('FULL', tostring(capacity), count)
end

local stateExists = redis.call('EXISTS', orderState) == 1
local queueLength = redis.call('LLEN', orderQueue)
local schedulerID = ''
if stateExists then
    local stateValues = redis.call('HMGET', orderState,
        'order_key', 'phase', 'depth', 'scheduler_id')
    local stateOrderKey = stateValues[1]
    local phase = stateValues[2]
    local depth = tonumber(stateValues[3] or '-1')
    schedulerID = stateValues[4] or ''
    if stateOrderKey ~= orderKey or phase == 'corrupt' or
       (phase ~= 'ready' and phase ~= 'owned') or depth ~= queueLength or
       depth <= 0 or schedulerID == '' then
        redis.call('HSET', orderState, 'phase', 'corrupt', 'corrupt_reason', 'state_conflict')
        redis.call('XADD', isolation, '*', 'reason', 'state_conflict',
            'order_key', orderKey, 'queue_length', tostring(queueLength))
        return result('STATE_CONFLICT', '', count)
    end
elseif queueLength > 0 then
    redis.call('XADD', isolation, '*', 'reason', 'orphan_list',
        'order_key', orderKey, 'queue_length', tostring(queueLength))
    return result('ISOLATED', '', count)
end

redis.call('RPUSH', orderQueue, payload)
count = redis.call('HINCRBY', partitionState, 'count', 1)

if not stateExists then
    local schedulerID = redis.call('XADD', scheduler, '*', 'order_key', orderKey)
    redis.call('HSET', orderState,
        'order_key', orderKey,
        'scheduler_id', schedulerID,
        'phase', 'ready',
        'consumer', '',
        'acquisition_id', '',
        'deadline_ms', '0',
        'depth', '1')
    return result('SCHEDULED', schedulerID, count)
end

redis.call('HINCRBY', orderState, 'depth', 1)
return result('QUEUED', schedulerID, count)
