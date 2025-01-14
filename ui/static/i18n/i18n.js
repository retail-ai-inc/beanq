const en_nav = [
    {
        "label":"Home",
        "to":"/admin/home",
        "sub":[]
    },
    {
        "label":"Schedule",
        "to":"/admin/schedule",
        "sub":[]
    },
    {
        "label":"Channel",
        "to":"/admin/queue",
        "sub":[]
    },
    {
        "label":"Log",
        "tos":["/admin/log/event","/admin/log/dlq","/admin/log/workflow"],
        "sub":[
            {"label":"Event Log","to":"/admin/log/event"},
            {"label":"DLQ Log","to":"/admin/log/dlq"},
            {"label":"WorkFlow Log","to":"/admin/log/workflow"}
        ]
    },
    {
        "label":"Redis",
        "tos":["/admin/redis","/admin/redis/monitor"],
        "sub":[
            {"label":"Info","to":"/admin/redis"},
            {"label":"Command","to":"/admin/redis/monitor"},
        ]
    }
]
const en_setting = [
    {
        "label":"Language",
        "tos":[],
        "sub":[
            {"label":"English","index":0, "to":""},
            {"label":"日本語 (Japanese)","index":1, "to":""},
        ]
    },
    {
        "label":"Setting",
        "tos":["/admin/optLog","/admin/user"],
        "sub":[
            {"label":"Operation Log","to":"/admin/optLog"},
            {"label":"User","to":"/admin/user"},
            {"label":"Logout","to":""},
        ]
    },
]
const en_retry_modal = {
    "title":"Are you sure to retry?",
    "body":"Trying again will not restore the data",
    "okButton":"Yes",
    "cancelButton":"cancel"
}
const en_delete_modal = {
    "title":"Are you sure to delete?",
    "body":"If you need to restore, please contact the administrator.",
    "okButton":"Yes",
    "cancelButton":"cancel"
}
const en_edit_modal = {
    "title":"Edit Payload",
    "okButton":"Edit",
    "cancelButton":"Close"
}
const jp_nav = [];
const jp_setting = [];
const jp_retry_modal = {};
const jp_delete_modal = {};
const jp_edit_modal = {};

const I18n = [
    {"key":"en","value":{
        "nav":en_nav,
        "setting":en_setting,
        "okButton":"Yes",
        "cancelButton":"cancel",
        "retryModal":en_retry_modal,
        "deleteModal":en_delete_modal,
        "editModal":en_edit_modal
    }},
    {"key":"jp","value":{
        "nav":jp_nav, // []
        "setting":jp_setting, // []
        "okButton":"Yes",
        "cancelButton":"cancel",
        "retryModal":jp_retry_modal, // {}
        "deleteModal":jp_delete_modal, // {}
        "editModal":jp_edit_modal // {}
    }}
]

let Lang = I18n[0].value;
let lang = sessionStorage.getItem("lang");
if(lang === "" || lang === "0"){
    Lang = I18n[0].value;
}else{
    I18n.forEach((v,k)=>{
        if(lang === v.key){
            Lang = v.value;
            return;
        }
    })
}
