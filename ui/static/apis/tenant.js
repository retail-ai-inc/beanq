const tenantApi = {
    List(page,pageSize,name,status){
        return request.get(`/tenant?page=${page}&pageSize=${pageSize}&name=${name}&status=${status}`);
    },
    Add(data){
        return request.put(`/tenant`,data,{
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            }
        });
    },
    Update(data){
        return request.post(`/tenant/update`,data);
    },
    Delete(id){
        return request.post(`/tenant/delete`,{id:id});
    },
    Get(id){
        return request.get(`/tenant/${id}`);
    }
}