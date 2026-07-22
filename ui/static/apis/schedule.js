const scheduleApi = {
    GetSchedule(page,pageSize){
		return request.get("schedules",{"params":{"page":page,"pageSize":pageSize}});
    }
}
