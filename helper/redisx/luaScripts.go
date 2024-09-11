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
const SaveHMSet = `

`
