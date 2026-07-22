const logApi = {

    OptLog(page,pageSize){
		return request.get(`operation-logs?page=${page}&pageSize=${pageSize}`);
    },
    DeleteOptLog(id){
	  return request.delete(`operation-logs/${id}`);
    },
    WorkFlowLogs(page,pageSize){
		return request.get(`workflow-logs?page=${page}&pageSize=${pageSize}`);
    }
}
