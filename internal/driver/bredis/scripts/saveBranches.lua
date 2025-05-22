-- LockGlobalSaveBranches
local old = redis.call('GET', KEYS[4])
if old ~= ARGV[3] then
	return 'NOT_FOUND'
end
local start = ARGV[4]
-- check duplicates for workflow
if start == "-1" then
	local t = cjson.decode(ARGV[5])
	local bs = redis.call('LRANGE', KEYS[2], 0, -1)
	for i = 1, table.getn(bs) do
		local c = cjson.decode(bs[i])
		if t['branch_id'] == c['branch_id'] and t['op'] == c['op'] then
			return 'UNIQUE_CONFLICT'
		end
	end
end

for k = 5, table.getn(ARGV) do
	if start == "-1" then
		redis.call('RPUSH', KEYS[2], ARGV[k])
	else
		redis.call('LSET', KEYS[2], start+k-5, ARGV[k])
	end
end
redis.call('EXPIRE', KEYS[2], ARGV[2])

return 0