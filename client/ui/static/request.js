const baseURL = "http://localhost:9090/";

const request = axios.create({
    baseURL:baseURL,
    timeout:5000,
    //responseType: 'json',
    responseEncoding: 'utf8',
})
request.interceptors.request.use(
    config=>{

        const token = sessionStorage.getItem("token");
        if(token){
            config.headers["BEANQ-Authorization"] = "Bearer " + token;
        }
        return config;
    },
    err=>{
        return Promise.reject(err);
    }
)
request.interceptors.response.use(
    res=>{
        let data = res.data;
        if (data.code == "0000"){
            return Promise.resolve(data);
        }
        return Promise.reject(new Error(data.msg));
    },
    err=>{

        if (err.response.status == 401){

            sessionStorage.clear();
        }
        return Promise.reject(err);
    }
)