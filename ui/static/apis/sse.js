const sseApi = {
    Init(url){
		if (typeof url !== "string" || url.trim() === "") {
			throw new TypeError("SSE URL must be a non-empty string");
		}
		return new EventSource(`/api/v1/${url}`, {withCredentials:true});
    }
}

if (typeof module !== "undefined" && module.exports) {
	module.exports = sseApi;
}
