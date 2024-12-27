const loginApi = {
    Login(username,password){
        return  request.post("login", {username:username,password:password} )
    }
}