const sequenceLockApi = {
    List(page,pageSize,orderKey,channelName,topicName){
        return request.get(`/sequenceLock/list?page=${page}&pageSize=${pageSize}&orderKey=${orderKey}&channelName=${channelName}&topicName=${topicName}`);
    },
    UnLock(orderKey){
        return request.delete(`/sequenceLock/unlock/${orderKey}`)
    }
}