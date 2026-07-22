const userApi = {
    List(page,pageSize,account){
		return request.get(`users?page=${page}&pageSize=${pageSize}&account=${account}`);
    },
    Add(data){
		return request.post("users",data);
    },
    Delete(id){
        let params = {id:id};
		return request.delete(`users/${params.id}`);
    },
    Edit(data){
		const {_id,...updates} = data;
		return request.patch(`users/${_id}`,updates);
    },
    Check(password){
		return request.post(`users/check-password`,{password:password})
    }
}
