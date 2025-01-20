const logApi = {

    OptLog(page,pageSize){
        return request.get(`/log/opt_log?page=${page}&pageSize=${pageSize}`);
    },
    DeleteOptLog(id){
      return request.delete(`/log/opt_log?id=${id}`);
    },
    WorkFlowLogs(page,pageSize){
        return request.get(`/log/workflow_log?page=${page}&pageSize=${pageSize}`);
    }
}