const workflowApi = {
    List(page, pageSize,channelName,topicName,status) {
		return request.get(`workflows?page=${page}&pageSize=${pageSize}&channel=${channelName}&topic=${topicName}&status=${status}`);
    },
    Delete(id) {
		return request.delete(`workflows/${id}`);
    },
}
