const dlqApi = {
    List(page,pageSize,id,status,moodType,topicName){
        return request.get(`/dlq/list?page=${page}&pageSize=${pageSize}&id=${id}&status=${status}&moodType=${moodType}&topicName=${topicName}`);
    },
    Delete(id){
        let params = {id:id};
        return request.post(`dlq/delete`,params);
    },
    Retry(id,data){
        return request.post(`/dlq/retry`,{uniqueId:id,data:JSON.stringify(data)});
    }
}