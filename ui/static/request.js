
const request = axios.create({
	baseURL:"/api/v1/",
	withCredentials:true,
	timeout:5000,
    //responseType: 'json',
    responseEncoding: 'utf8',
})
request.interceptors.request.use(
    config=>{

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
		if ([401, 403].includes(err?.response?.status) && !isLoginRequest(err.config)) {
			redirectToLogin();
		}
		return Promise.reject(err);
    }
)

function isLoginRequest(config) {
	return typeof config?.url === "string" && config.url.replace(/^\//, "").endsWith("auth/login");
}

let redirectingToLogin = false;

function redirectToLogin() {
	if (redirectingToLogin || window.location.hash === "#/login") {
		return;
	}
	redirectingToLogin = true;
	Storage.Clear();
	window.location.replace("/#/login");
}
