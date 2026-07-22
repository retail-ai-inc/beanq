const dlqApi = {
    List(page,pageSize,id,status,moodType,topicName){
		return request.get(`dead-letters?page=${page}&pageSize=${pageSize}&id=${id}&status=${status}&moodType=${moodType}&topicName=${topicName}`);
    },
    Delete(id){
		return request.delete(`dead-letters/${id}`);
    },
    Retry(id,data){
		return request.post(`dead-letters/${id}/retry`,{data:data});
    }
}
