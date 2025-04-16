const configApi = {
    // Get configuration
    getConfig(){
        return request.get("/redis/config");
    },
    // Update Configuration
    updateConfig(data){
        return request.put("/redis/config",data,{
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            }
        });
    },
 }