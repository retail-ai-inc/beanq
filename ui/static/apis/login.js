const loginApi = {
    Login(username,password,expiredTimeBool, recaptchaToken){
        let expiredTime = 0
        if(expiredTimeBool){
            expiredTime = 30
        }
		return request.post("auth/login", {username:username,password:password,expiredDays:expiredTime,recaptchaToken: recaptchaToken || ""}, {timeout: 20000})
    },
    AllowGoogle(){
		return request.get("auth/google/config")
    },
}
