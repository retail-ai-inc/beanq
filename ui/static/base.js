
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
    GetLang(i18n){
        let ls = Storage.GetItem("lang") || "0";
        let lang = parseInt(ls);
        return i18n[lang].value;
    },
    Alert(message,type){
        const alertPlaceholder = document.getElementById('payloadAlertInfo');
        alertPlaceholder.innerHTML = `<div class="alert alert-${type} alert-dismissible" id="my-alert" role="alert">
          <div>${message}</div>
          <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
          </div>`;
    },
    CheckEmail(email){
        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        return emailRegex.test(email);
    },
    MaskString(str, start=3, end=4, mask = '*') {
        const startPart = _.take(str.split(''), start).join('');
        const endPart = _.takeRight(str.split(''), end).join('');
        const middleLength = (str.length - start - end) > 4 ? 4 :str.length-start-end;
        const masked = _.repeat(mask, middleLength);
        return startPart + masked + endPart;
    },
    FormatRelativeTime(pastTime){
        const now = new Date();
        const past = new Date(pastTime);
        const diffMs = now - past;

        const seconds = Math.floor(diffMs / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);
        const weeks = Math.floor(days / 7);

        if (diffMs < 0) {
            return "future time";
        }

        if (seconds < 60) {
            return `${seconds} seconds ago`;
        } else if (minutes < 60) {
            return `${minutes} minutes ago`;
        } else if (hours < 24) {
            return `${hours} hours ago`;
        } else if (days < 7) {
            return `${days} days ago`;
        } else if (weeks < 4) {
            return `${weeks} weeks ago`;
        } else {
            // More than one month will display the complete date
            return past.toLocaleDateString("zh-CN", {
                year: "numeric",
                month: "2-digit",
                day: "2-digit"
            });
        }
    }
}

const Storage = {
    SetItem(key,val){
        return localStorage.setItem(key,val);
    },
    GetItem(key){
        return localStorage.getItem(key);
    },
    Clear(){
        return localStorage.clear();
    }
}