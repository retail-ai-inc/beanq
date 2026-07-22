const configApi = {
    // Get configuration
    getConfig(){
		return request.get("config");
    },
    // Update Configuration
    updateConfig(data){
		return request.put("config",data,{
            headers: {
                'Content-Type': 'application/json',
                'Accept': 'application/json'
            }
        });
    },
 }
