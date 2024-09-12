package lua

const (
	HashDuplicateId = `
	local key = KEYS[1]

	local result = redis.pcall('HEXISTS',key,'id')

	if type(result) == 'table' and result.err then
		return {err = result.err}
	end

	if result == 1 then
		local hresult = redis.pcall('HINCRBY',key,'score',1)
		if type(hresult) == 'table' and hresult.err then
			return {err = hresult.err}
		end
		return {err = "duplicate id"}
	end

	return true
`

	SaveHSet = `
	local key = KEYS[1]
	local data = ARGV

	for i = 1, #data, 2 do
		local field = data[i]
		local value = data[i + 1]
		redis.call('HSET', key, field, value)
	end

	local ttl = 3600*tonumber(KEYS[2])
	redis.call('EXPIRE',key,ttl)

	return true
`
)