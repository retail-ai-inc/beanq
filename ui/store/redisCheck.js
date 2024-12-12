import request  from "request";

export const healthStore = defineStore("health",{
    state:()=>({
        health:{}
    }),
    getters:{
        getHealthData:async (state)=>{
            let data = await request.get("redis/sfsf");
            Object.assign(state.health,data.data);
        }
    }
})