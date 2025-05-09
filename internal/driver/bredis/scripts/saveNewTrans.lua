-- MaySaveNewTrans
local g = redis.call('GET', KEYS[1])
if g ~= false then
	return 'UNIQUE_CONFLICT'
end

redis.call('SET', KEYS[1], ARGV[3], 'EX', ARGV[2])
redis.call('SET', KEYS[4], ARGV[6], 'EX', ARGV[2])
redis.call('ZADD', KEYS[3], ARGV[4], ARGV[5])
for k = 7, table.getn(ARGV) do
	redis.call('RPUSH', KEYS[2], ARGV[k])
end
redis.call('EXPIRE', KEYS[2], ARGV[2])