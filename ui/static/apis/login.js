const loginApi = {
    Login(username,password,expiredTimeBool){
        let expiredTime = 0
        if(expiredTimeBool){
            expiredTime = 30
        }
        return  request.post("login", {username:username,password:password,expiredTime:expiredTime} )
    },
    AllowGoogle(){
        return request.get("login/allowGoogle")
    },
}