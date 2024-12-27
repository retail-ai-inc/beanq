const logApi = {

    OptLog(page,pageSize){
        return request.get(`/log/opt_log?page=${page}&pageSiz=${pageSize}`);
    }
}