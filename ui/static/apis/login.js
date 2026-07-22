const loginApi = {
    Login(username,password,expiredTimeBool){
        let expiredTime = 0
        if(expiredTimeBool){
            expiredTime = 30
        }
		return request.post("auth/login", {username:username,password:password,expiredDays:expiredTime})
    },
    AllowGoogle(){
		return request.get("auth/google/config")
    },
}
