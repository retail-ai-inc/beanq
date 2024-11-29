const sseApi = {
    Init(url){
        let token = sessionStorage.getItem("token");
        let urlEs ;
        if(url.includes("?")){
            urlEs = `${url}&token=${token}`;
        }else{
            urlEs = `${url}?token=${token}`;
        }
        return new EventSource(urlEs);
    }
}