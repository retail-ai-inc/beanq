const eventApi = {
    // make bootstrap alert html element
    Alert(message,type){
        const alertPlaceholder = document.getElementById('payloadAlertInfo');
        alertPlaceholder.innerHTML = `<div class="alert alert-${type} alert-dismissible" id="my-alert" role="alert">
      <div>${message}</div>
      <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
      </div>`;
    },
    Delete(id){
		return request.delete(`events/${id}`);
    },
    Edit(id,payload){
		return request.patch(`events/${id}`,{payload:payload});
    },
    Retry(id,data){
		return request.post(`events/${id}/retry`,{data:data});
    }
}
