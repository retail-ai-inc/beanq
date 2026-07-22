const tenantApi = {
    List(page,pageSize,name,status){
		return request.get(`tenants?page=${page}&pageSize=${pageSize}&name=${name}&status=${status}`);
    },
    Add(data){
		return request.post(`tenants`,data,{
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            }
        });
    },
    Update(id,data){
		return request.patch(`tenants/${id}`,data,{
            headers:{
                'Content-Type':'application/json',
                'Accept':'application/json'
            }
        });
    },
    Delete(id){
		return request.delete(`tenants/${id}`);
    },
    Get(id){
		return request.get(`tenants/${id}`);
    }
}
