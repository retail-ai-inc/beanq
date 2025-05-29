const workflowApi = {
    List(page, pageSize,channelName,topicName,status) {
        return request.get(`/workflow/list?page=${page}&pageSize=${pageSize}&channel=${channelName}&topic=${topicName}&status=${status}`);
    },
    Delete(id) {
        let params = {id: id};
        return request.post(`/workflow/delete`, params);
    },
}