const sseApi = {
    Init(url){
        let token = sessionStorage.getItem("token");
        let nodeId = sessionStorage.getItem("nodeId");
        let urlEs ;
        if(url.includes("?")){
            urlEs = `${url}&token=${token}&nodeId=${nodeId}`;
        }else{
            urlEs = `${url}?token=${token}&nodeId=${nodeId}`;
        }
        return new EventSource(urlEs);
    }
}