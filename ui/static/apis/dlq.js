const dlqApi = {
    List(page,pageSize){
        return request.get(`/dlq/list?page=${page}&pageSize=${pageSize}`);
    },
    Delete(id){
        let params = {id:id};
        return request.post(`dlq/delete`,params);
    },
    Retry(id,data){
        return request.post(`/dlq/retry`,{uniqueId:id,data:JSON.stringify(data)});
    }
}