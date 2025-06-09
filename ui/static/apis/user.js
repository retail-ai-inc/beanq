const userApi = {
    List(page,pageSize,account){
        return request.get(`/user/list?page=${page}&pageSize=${pageSize}&account=${account}`);
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
    },
    Check(password){
        return request.post(`/user/check`,{password:password})
    }
}