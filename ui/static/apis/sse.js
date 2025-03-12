const sseApi = {
    Init(url){

        let token = Storage.GetItem("token");
        let nodeId = Storage.GetItem("nodeId");

        let urlEs ;
        if(url.includes("?")){
            urlEs = `${url}&token=${token}&nodeId=${nodeId}`;
        }else{
            urlEs = `${url}?token=${token}&nodeId=${nodeId}`;
        }
        return new EventSource(urlEs);
    }
}