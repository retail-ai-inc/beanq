const userApi = {
    List(page,pageSize){
        return request.get(`/user/list?page=${page}&pageSize=${pageSize}`);
    },
    Add(data){
        return request.post("/user/add",data);
    },
    Delete(id){
        let params = {id:id};
        return request.post(`/user/del`,params);
    },
    Edit(data){
        return request.post(`/user/edit`,data);
    }
}