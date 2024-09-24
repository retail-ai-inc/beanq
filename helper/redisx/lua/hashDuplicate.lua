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
    return 1
end

return 0