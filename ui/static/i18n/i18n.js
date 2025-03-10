const Langs = [
    {label:"English",index:0, flag:"en"},
    {label:"日本語 (Japanese)",index:1, flag:"ja"},
];

const Nav = [
    {id:1,label:"Home",mark:"home",en:"Home",jp:"ホーム",to:"/admin/home",children:[],value:"/dashboard",pid:0},
    {id:2,label:"Schedule",mark:"schedule",en:"Schedule",jp:"スケジュール",to:"/admin/schedule",children:[],value:"/schedule",pid:0},
    {id:3,label:"Channel",mark:"channel",en:"Channel",jp:"チャンネル",to:"/admin/queue",children:[],value:"/queue/list",pid:0},
    {id:4,label:"Log",mark:"log",en:"Log",jp:"ログ",to:"/admin/home",tos:["/admin/log/event","/admin/log/dlq","/admin/log/workflow"],value:"log",pid:0,children:[
            {id:5,label:"Event Log",mark:"evengLog",en:"Event Log",jp:"イベントログ",to:"/admin/log/event",value:"/event_log/list",pid:4,children:[
                    {id:6,label:"Edit",mark:"edit",en:"Edit",jp:"編集",value:"/event_log/edit",pid:5},
                    {id:7,label:"Delete",mark:"delete",en:"Delete",jp:"削除する",value:"/event_log/delete",pid:5},
                    {id:8,label:"Retry",mark:"retry",en:"Retry",jp:"再試行",value:"/event_log/retry",pid:5},
                ]
            },
            {id:9,label:"DLQ Log",mark:"dlqLog",en:"DLQ Log",jp:"DLQ Log",to:"/admin/log/dlq",value:"",pid:4,children: [
                    {id:10,label:"Edit",mark:"edit",en:"Edit",jp:"編集",value:"",pid:9},
                    {id:11,label:"Delete",mark:"delete",en:"Delete",jp:"削除する",value:"",pid:9},
                    {id:12,label:"Retry",mark:"retry",en:"Retry",jp:"再試行",value:"",pid:9}
                ]
            },
            {id:13,label: "Workflow Log",mark:"workflowLog",en:"Workflow Log",jp:"Workflow Log",to:"/admin/log/workflow",value:"",pid:4,children: [
                    {id:14,label:"Edit",mark:"edit",en:"Edit",jp:"編集",value:"",pid:13},
                    {id:15,label:"Delete",mark:"delete",en:"Delete",jp:"削除する",value:"",pid:13},
                    {id:16,label:"Retry",mark:"retry",en:"Retry",jp:"再試行",value:"",pid:13}
                ]
            }
        ]
    },
    {id:17,label:"Redis",mark:"redis",en:"Redis",jp:"Redis",value:"",tos:["/admin/redis","/admin/redis/monitor"],pid:0,children: [
            {id:18,label:"Info",mark:"info",en:"Info",jp:"情報",to:"/admin/redis",value:"/redis",pid:17},
            {id:19,label:"Command",mark:"command",en:"Command",jp:"指揮部",to:"/admin/redis/monitor",value:"/redis/monitor",pid:17}
        ]
    },
    {id:20,label:"Setting",mark:"setting",en:"Setting",jp:"設定",tos:["/admin/optLog","/admin/user"],value:"",pid:0,children: [
            {id:21,label:"Operation Log",mark:"operationLog",en:"Operation Log",jp:"Operation Log",to:"/admin/optLog",value:"/log/opt_log",pid:20,children: [
                    {id:30,label:"Delete",mark:"delete",en:"Delete",jp:"削除する",value:"/log/opt_log",pid:21}
                ]
            },
            {id:22,label:"User",mark:"user",en:"User",jp:"ユーザー",to:"/admin/user",value:"/user/list",pid:20,children: [
                    {id:23,label:"Add",mark:"add",en:"Add",jp:"追加",value:"/user/add",pid:22},
                    {id:24,label:"Delete",mark:"delete",en:"Delete",jp:"削除",value:"/user/del",pid:22},
                    {id:25,label:"Edit",mark:"edit",en:"Edit",jp:"編集",value:"/user/edit",pid:22}
                ]
            },
            {id:26,label:"Role",mark:"role",en:"Role",jp:"Role",to:"/admin/role",value:"",pid:20,children: [
                    {id:27,label:"Add",mark:"add",en:"Add",jp:"追加",value:"",pid:26},
                    {id:28,label:"Delete",mark:"delete",en:"Delete",jp:"削除する",value:"",pid:26},
                    {id:29,label:"Edit",mark:"edit",en:"Edit",jp:"編集",value:"",pid:26}
                ]
            }
        ]
    }
];

