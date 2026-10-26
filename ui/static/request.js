
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

const LOGIN_REDIRECT_KEY = "beanq_login_redirect";
const DEFAULT_AFTER_LOGIN = "/admin/home";

function currentRoutePath() {
	return (window.location.hash || "").replace(/^#/, "") || "/";
}

function isLoginHash(hash) {
	const path = (hash || "").replace(/^#/, "");
	return path === "/login" || path.startsWith("/login?");
}

function isSafeLoginRedirect(path) {
	if (typeof path !== "string") {
		return false;
	}
	const value = path.trim();
	if (!value.startsWith("/") || value.startsWith("//") || value.startsWith("/\\")) {
		return false;
	}
	if (value.includes("://") || value.includes("\\")) {
		return false;
	}
	const pathname = value.split("?")[0].split("#")[0];
	if (pathname === "/login" || pathname.startsWith("/login/")) {
		return false;
	}
	return pathname === "/admin" || pathname.startsWith("/admin/");
}

function rememberLoginRedirect(path) {
	const redirect = isSafeLoginRedirect(path) ? path.trim() : "";
	if (!redirect) {
		return "";
	}
	try {
		sessionStorage.setItem(LOGIN_REDIRECT_KEY, redirect);
	} catch {}
	return redirect;
}

function takeLoginRedirect(queryRedirect) {
	let candidate = queryRedirect;
	if (!isSafeLoginRedirect(candidate)) {
		try {
			candidate = sessionStorage.getItem(LOGIN_REDIRECT_KEY);
		} catch {
			candidate = "";
		}
	}
	try {
		sessionStorage.removeItem(LOGIN_REDIRECT_KEY);
	} catch {}
	return isSafeLoginRedirect(candidate) ? candidate.trim() : DEFAULT_AFTER_LOGIN;
}

let redirectingToLogin = false;

function redirectToLogin() {
	if (redirectingToLogin || isLoginHash(window.location.hash)) {
		return;
	}
	redirectingToLogin = true;
	const redirect = rememberLoginRedirect(currentRoutePath());
	Storage.Clear();
	const loginHash = redirect
		? "/#/login?redirect=" + encodeURIComponent(redirect)
		: "/#/login";
	window.location.replace(loginHash);
}
