const tenantApi = {
    List(page,pageSize,name,status){
        return request.get(`/tenant?page=${page}&pageSize=${pageSize}&name=${name}&status=${status}`);
    },
    Add(data){
        return request.post(`/tenant`,data,{
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            }
        });
    },
    Update(id,data){
        return request.put(`/tenant/${id}`,data,{
            headers:{
                'Content-Type':'application/json',
                'Accept':'application/json'
            }
        });
    },
    Delete(id){
        return request.delete(`/tenant/${id}`);
    },
    Get(id){
        return request.get(`/tenant/${id}`);
    }
}