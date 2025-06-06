const Langs = [
    {label:"English",index:0, flag:"en"},
    {label:"日本語 (Japanese)",index:1, flag:"ja"},
];

const Nav = [
    {id:1,label:"Home",mark:"home",to:"/admin/home",children:[],value:"/dashboard",pid:0},
    {id:2,label:"Schedule",mark:"schedule",to:"/admin/schedule",children:[],value:"/schedule",pid:0},
    {id:3,label:"Channel",mark:"channel",to:"/admin/queue",children:[],value:"/queue/list",pid:0},
    {id:4,label:"Log",mark:"log",to:"/admin/home",tos:["/admin/log/event","/admin/log/dlq","/admin/log/workflow"],value:"log",pid:0,children:[
            {id:5,label:"Event Log",mark:"evengLog",to:"/admin/log/event",value:"/event_log/list",pid:4,children:[
                    {id:6,label:"Edit",mark:"edit",value:"/event_log/edit",pid:5},
                    {id:7,label:"Delete",mark:"delete",value:"/event_log/delete",pid:5},
                    {id:8,label:"Retry",mark:"retry",value:"/event_log/retry",pid:5},
                ]
            },
            {id:9,label:"DLQ Log",mark:"dlqLog",to:"/admin/log/dlq",value:"",pid:4,children: [
                    {id:10,label:"Edit",mark:"edit",value:"",pid:9},
                    {id:11,label:"Delete",mark:"delete",value:"",pid:9},
                    {id:12,label:"Retry",mark:"retry",value:"",pid:9}
                ]
            },
            {id:13,label: "Workflow Log",mark:"workflowLog",to:"/admin/log/workflow",value:"",pid:4,children: [
                    {id:14,label:"Edit",mark:"edit",value:"",pid:13},
                    {id:15,label:"Delete",mark:"delete",value:"",pid:13},
                    {id:16,label:"Retry",mark:"retry",value:"",pid:13}
                ]
            },
            {id:32,label: "Sequence Lock",mark:"sequenceLock",to:"/admin/log/sequence_lock",value:"",pid:4,children: [
                    {id:33,label:"Unlock",mark:"unlock",value:"",pid:32},
                ]
            }
        ]
    },
    {id:17,label:"Redis",mark:"redis",value:"",tos:["/admin/redis","/admin/redis/monitor"],pid:0,children: [
            {id:18,label:"Info",mark:"info",to:"/admin/redis",value:"/redis",pid:17},
            {id:19,label:"Command",mark:"command",to:"/admin/redis/monitor",value:"/redis/monitor",pid:17}
        ]
    },
    {id:20,label:"Setting",mark:"setting",tos:["/admin/optLog","/admin/user","/admin/role","/admin/config"],value:"",pid:0,children: [
            {id:21,label:"Operation Log",mark:"operationLog",to:"/admin/optLog",value:"/log/opt_log",pid:20,children: [
                    {id:30,label:"Delete",mark:"delete",value:"/log/opt_log",pid:21}
                ]
            },
            {id:22,label:"User",mark:"user",to:"/admin/user",value:"/user/list",pid:20,children: [
                    {id:23,label:"Add",mark:"add",value:"/user/add",pid:22},
                    {id:24,label:"Delete",mark:"delete",value:"/user/del",pid:22},
                    {id:25,label:"Edit",mark:"edit",value:"/user/edit",pid:22}
                ]
            },
            {id:26,label:"Role",mark:"role",to:"/admin/role",value:"",pid:20,children: [
                    {id:27,label:"Add",mark:"add",value:"",pid:26},
                    {id:28,label:"Delete",mark:"delete",value:"",pid:26},
                    {id:29,label:"Edit",mark:"edit",value:"",pid:26}
                ]
            },
            {id:31,label:"Config",mark:"config",to:"/admin/config",value:"/admin/config",pid:20}
        ]
    }
];

