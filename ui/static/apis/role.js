const roleApi = {
    List(page,pageSize){
        return request.get(`/role/list?page=${page}&pageSize=${pageSize}`);
    },
    Delete(id){
        let params = {id:id};
        return request.post(`/role/delete`,params);
    },
    Edit(id,data){
        let ndata = {};
        ndata._id = id;
        ndata.roles = JSON.stringify(data.roles);
        return request.post(`/role/edit`,ndata);
    },
    Add(data){
        let ndata = {};
        ndata.name = data.name;
        ndata.roles = JSON.stringify(data.roles);
        return request.post("/role/add",ndata);
    },
}