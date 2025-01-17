
const Base = {
    Debounce(func, wait) {
        let timeout;
        return function (...args) {
            clearTimeout(timeout);
            timeout = setTimeout(() => {
                func.apply(this, args);
            }, wait);
        };
    },
    Alert(message,type){
        const alertPlaceholder = document.getElementById('payloadAlertInfo');
        alertPlaceholder.innerHTML = `<div class="alert alert-${type} alert-dismissible" id="my-alert" role="alert">
          <div>${message}</div>
          <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
          </div>`;
    }
}