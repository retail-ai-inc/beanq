package redisx

const HashDuplicateIdLua = `
	local key = KEYS[1]

	local result,err = redis.pcall('HEXISTS',key,'id')
	if not result then 
		return err
	end

	if result == 1 then
		local hresult,herr = redis.pcall('HINCRBY',key,'score',1)
		if not hresult then
			return herr
		end
		return {err = "duplicate id"}
	end

	return {success = true}
`
const SaveHSet = `
	local key = KEYS[1]
	local data = ARGV

	for i = 1, #data, 2 do
		local field = data[i]
		local value = data[i + 1]
		redis.call('HSET', key, field, value)
	end

	local ttl = 3600*6
	redis.call('EXPIRE',key,ttl)

	return true
`
