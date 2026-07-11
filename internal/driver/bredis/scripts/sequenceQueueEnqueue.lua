local customerQueue = KEYS[1]
local customerState = KEYS[2]
local scheduler = KEYS[3]
local pending = KEYS[4]

local payload = ARGV[1]
local customerID = ARGV[2]
local maxPending = tonumber(ARGV[3])

if maxPending == nil or maxPending <= 0 then
    return {'BAD_CAPACITY'}
end

local count = tonumber(redis.call('GET', pending) or '0')
if count >= maxPending then
    return {'FULL', tostring(count)}
end

local stateExists = redis.call('EXISTS', customerState)
if stateExists == 1 then
    local stateCustomer = redis.call('HGET', customerState, 'customer_id')
    if stateCustomer ~= customerID then
        return {'STATE_CONFLICT'}
    end
end

redis.call('RPUSH', customerQueue, payload)
local nextCount = redis.call('INCR', pending)

if stateExists == 0 then
    local schedulerID = redis.call('XADD', scheduler, '*', 'customer_id', customerID)
    redis.call('HSET', customerState,
        'customer_id', customerID,
        'scheduler_id', schedulerID,
        'phase', 'ready',
        'owner', '',
        'consumer', '',
        'deadline_ms', '0')
    return {'SCHEDULED', schedulerID, tostring(nextCount)}
end

return {'QUEUED', redis.call('HGET', customerState, 'scheduler_id'), tostring(nextCount)}
