
axios.defaults.baseURL = "./"
axios.defaults.headers.post["Content-Type"] = "multipart/form-data";

const request = axios.create({
    timeout:5000,
    //responseType: 'json',
    responseEncoding: 'utf8',
})
request.interceptors.request.use(
    config=>{

        const token = Storage.GetItem("token");
        if(token){
            config.headers["BEANQ-Authorization"] = "Bearer " + token;
        }
        config.headers["X-Cluster-Nodeid"] = Storage.GetItem("nodeId");
        config.headers["X-Role-Id"] = Storage.GetItem("roleId");
        return config;
    },
    err=>{
        return Promise.reject(err);
    }
)
request.interceptors.response.use(
    res=>{
        const {code,msg,data} = res.data;
        if (code === "0000"){
            return Promise.resolve(data);
        }
        return Promise.reject(new Error(msg));
    },
    err=>{
        console.log("request err",err)
        return Promise.reject(err);
    }
)