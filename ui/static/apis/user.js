const userApi = {
    List(){
        return request.get("/user/list");
    },
    Add(data){
        return request.post("/user/add",data);
    },
    Delete(account){
        let params = {account:account};
        return request.post(`/user/del`,params);
    },
    Edit(data){
        return request.post(`/user/edit`,data);
    }
}