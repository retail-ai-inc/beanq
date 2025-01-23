const Langs = [
    {"label":"English","index":0, "to":""},
    {"label":"日本語 (Japanese)","index":1, "to":""},
];


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
        "sub":Langs
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
const jp_nav = [
    {
        "label":"ホーム",
        "to":"/admin/home",
        "sub":[]
    },
    {
        "label":"スケジュール",
        "to":"/admin/schedule",
        "sub":[]
    },
    {
        "label":"チャンネル",
        "to":"/admin/queue",
        "sub":[]
    },
    {
        "label":"ログ",
        "tos":["/admin/log/event","/admin/log/dlq","/admin/log/workflow"],
        "sub":[
            {"label":"イベントログ","to":"/admin/log/event"},
            {"label":"DLQ ログ","to":"/admin/log/dlq"},
            {"label":"ワークフローログ","to":"/admin/log/workflow"}
        ]
    },
    {
        "label":"Redis",
        "tos":["/admin/redis","/admin/redis/monitor"],
        "sub":[
            {"label":"情報","to":"/admin/redis"},
            {"label":"コマンド","to":"/admin/redis/monitor"},
        ]
    }
];
const jp_setting = [
    {
        "label":"Language",
        "tos":[],
        "sub":Langs
    },
    {
        "label":"設定",
        "tos":["/admin/optLog","/admin/user"],
        "sub":[
            {"label":"操作ログ","to":"/admin/optLog"},
            {"label":"ユーザー","to":"/admin/user"},
            {"label":"ログアウト","to":""},
        ]
    },
];
const jp_retry_modal = {
    "title":"あなたは確かに再試すか?",
    "body":"再度試すと、データは復元されません。",
    "okButton":"はい",
    "cancelButton":"キャンセル"
};
const jp_delete_modal = {
    "title":"削除したいと思いますか？",
    "body":"リストアが必要な場合は、管理者に連絡してください。",
    "okButton":"はい",
    "cancelButton":"キャンセル"
};
const jp_edit_modal = {
    "title":"ペイロードの編集",
    "okButton":"編集",
    "cancelButton":"閉じる"
};

const I18n = [
    {"key":"en","value":{
        "nav":en_nav,
        "setting":en_setting,

        "okButton":"Yes",
        "cancelButton":"cancel",
        "editButton":"Edit",
        "addButton":"Add",
        "closeButton":"Close",

        "retryModal":en_retry_modal,
        "deleteModal":en_delete_modal,
        "editModal":en_edit_modal,
        "search":"Search"
    }},
    {"key":"jp","value":{
        "nav":jp_nav, // []
        "setting":jp_setting, // []

        "okButton":"はい",
        "cancelButton":"キャンセル",
        "editButton":"編集",
        "addButton":"追加",
        "closeButton":"閉じる",

        "retryModal":jp_retry_modal, // {}
        "deleteModal":jp_delete_modal, // {}
        "editModal":jp_edit_modal, // {}
        "search":"検索"
    }}
]
