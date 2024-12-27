local key = KEYS[1]
local streamKey = KEYS[2]
local fields = ARGV

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

local message = {}
for i = 1, #fields, 2 do
    table.insert(message, fields[i])
    table.insert(message, fields[i+1])
end

local id = redis.call('XADD', streamKey, '*', unpack(message))

return 0