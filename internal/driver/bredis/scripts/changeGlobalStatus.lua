-- ChangeGlobalStatus
local old = redis.call('GET', KEYS[4])
if old ~= ARGV[4] then
  return 'NOT_FOUND'
end
redis.call('SET', KEYS[1],  ARGV[3], 'EX', ARGV[2])
redis.call('SET', KEYS[4],  ARGV[7], 'EX', ARGV[2])
if ARGV[5] == '1' then
	redis.call('ZREM', KEYS[3], ARGV[6])
	redis.call('EXPIRE', KEYS[1], ARGV[8])
	redis.call('EXPIRE', KEYS[2], ARGV[8])
	redis.call('EXPIRE', KEYS[4], ARGV[8])
end

return 0