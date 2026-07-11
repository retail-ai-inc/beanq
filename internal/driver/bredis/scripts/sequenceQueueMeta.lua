local metadata = KEYS[1]
local schemaVersion = ARGV[1]
local partitions = ARGV[2]

if redis.call('EXISTS', metadata) == 0 then
    redis.call('HSET', metadata,
        'schema_version', schemaVersion,
        'partitions', partitions)
    return {'CREATED'}
end

local actualVersion = redis.call('HGET', metadata, 'schema_version')
local actualPartitions = redis.call('HGET', metadata, 'partitions')
if actualVersion ~= schemaVersion or actualPartitions ~= partitions then
    return {'MISMATCH', actualVersion or '', actualPartitions or ''}
end

return {'OK'}
