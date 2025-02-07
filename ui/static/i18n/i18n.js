const Langs = [
    {label:"English",index:0, to:""},
    {label:"日本語 (Japanese)",index:1, to:""},
];


const en_nav = [
    {
        label:"Home",
        flag:"Home",
        to:"/admin/home",
        sub:[]
    },
    {
        label:"Schedule",
        flag:"Schedule",
        to:"/admin/schedule",
        sub:[]
    },
    {
        label:"Channel",
        flag:"Channel",
        to:"/admin/queue",
        sub:[]
    },
    {
        label:"Log",
        flag:"Log",
        tos:["/admin/log/event","/admin/log/dlq","/admin/log/workflow"],
        sub:[
            {label:"Event Log",flag:"Event",to:"/admin/log/event"},
            {label:"DLQ Log",flag:"DLQ",to:"/admin/log/dlq"},
            {label:"Workflow Log",flag:"Workflow",to:"/admin/log/workflow"}
        ]
    },
    {
        label:"Redis",
         flag:"Redis",
        tos:["/admin/redis","/admin/redis/monitor"],
        sub:[
            {label:"Info",flag:"Info",to:"/admin/redis"},
            {label:"Command",flag:"Command",to:"/admin/redis/monitor"},
        ]
    }
]
const en_setting = [
    {
        label:"Language",
        flag:"Language",
        tos:[],
        sub:Langs
    },
    {
        label:"Setting",
        flag:"Setting",
        tos:["/admin/optLog","/admin/user"],
        sub:[
            {label:"Operation Log",flag:"Operation",to:"/admin/optLog"},
            {label:"Role",flag:"Role",to:"/admin/role"},
            {label:"User",flag:"User",to:"/admin/user"},
            {label:"Logout",flag:"Logout",to:""},
        ]
    },
]
const en_retry_modal = {
    title:"Are you sure to retry?",
    body:"Trying again will not restore the data",
    okButton:"Yes",
    cancelButton:"cancel"
}
const en_delete_modal = {
    title:"Are you sure to delete?",
    body:"If you need to restore, please contact the administrator.",
    okButton:"Yes",
    cancelButton:"cancel"
}
const en_edit_modal = {
    title:"Edit Payload",
    okButton:"Edit",
    cancelButton:"Close"
}
const jp_nav = [
    {
        label:"ホーム",
        flag:"Home",
        to:"/admin/home",
        sub:[]
    },
    {
        label:"スケジュール",
        flag:"Schedule",
        to:"/admin/schedule",
        sub:[]
    },
    {
        label:"チャンネル",
        flag:"Channel",
        to:"/admin/queue",
        sub:[]
    },
    {
        label:"ログ",
        flag:"Log",
        tos:["/admin/log/event","/admin/log/dlq","/admin/log/workflow"],
        sub:[
            {label:"イベントログ",flag:"Event",to:"/admin/log/event"},
            {label:"DLQ ログ",flag:"DLQ",to:"/admin/log/dlq"},
            {label:"ワークフローログ",flag:"Workflow",to:"/admin/log/workflow"}
        ]
    },
    {
        label:"Redis",
        flag:"Redis",
        tos:["/admin/redis","/admin/redis/monitor"],
        sub:[
            {label:"情報",flag:"Info",to:"/admin/redis"},
            {label:"コマンド",flag:"Command",to:"/admin/redis/monitor"},
        ]
    }
];
const jp_setting = [
    {
        label:"Language",
        flag:"Language",
        tos:[],
        sub:Langs
    },
    {
        label:"設定",
        flag:"Setting",
        tos:["/admin/optLog","/admin/user"],
        sub:[
            {label:"操作ログ",flag:"Operation",to:"/admin/optLog"},
            {label:"役割",flag:"Role",to:"/admin/role"},
            {label:"ユーザー",flag:"User",to:"/admin/user"},
            {label:"ログアウト",flag:"Logout",to:""},
        ]
    },
];
const jp_retry_modal = {
    title:"あなたは確かに再試すか?",
    body:"再度試すと、データは復元されません。",
    okButton:"はい",
    cancelButton:"キャンセル"
};
const jp_delete_modal = {
    title:"削除したいと思いますか？",
    body:"リストアが必要な場合は、管理者に連絡してください。",
    okButton:"はい",
    cancelButton:"キャンセル"
};
const jp_edit_modal = {
    title:"ペイロードの編集",
    okButton:"編集",
    cancelButton:"閉じる"
};

const I18n = [
    {key:"en",value:{
        nav:en_nav,
        setting:en_setting,

        okButton:"Yes",
        cancelButton:"cancel",
        editButton:"Edit",
        addButton:"Add",
        closeButton:"Close",

        retryModal:en_retry_modal,
        deleteModal:en_delete_modal,
        editModal:en_edit_modal,
        search:"Search"
    }},
    {key:"jp",value:{
        nav:jp_nav, // []
        setting:jp_setting, // []

        okButton:"はい",
        cancelButton:"キャンセル",
        editButton:"編集",
        addButton:"追加",
        closeButton:"閉じる",

        retryModal:jp_retry_modal, // {}
        deleteModal:jp_delete_modal, // {}
        editModal:jp_edit_modal, // {}
        search:"検索"
    }}
]
