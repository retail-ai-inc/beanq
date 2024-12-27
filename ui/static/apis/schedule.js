const scheduleApi = {
    GetSchedule(page,pageSize){
        return request.get("schedule",{"params":{"page":page,"pageSize":pageSize}});
    }
}