const workflowApi = {
    List(page, pageSize) {
        return request.get(`/workflow/list?page=${page}&pageSize=${pageSize}`);
    },
    Delete(id) {
        let params = {id: id};
        return request.post(`/workflow/delete`, params);
    },
}