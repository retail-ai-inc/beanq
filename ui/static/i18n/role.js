const role = [
    {id:1,label:"Home",value:"/dashboard",pid:0},
    {id:2,label:"Schedule",value:"/schedule",pid:0},
    {id:3,label:"Channel",value:"/queue/list",pid:0},
    {id:4,label:"Log",value:"log",pid:0,children:[
            {id:5,label:"Event Log",value:"/event_log/list",pid:4,children:[
                    {id:6,label:"Edit",value:"/event_log/edit",pid:5},
                    {id:7,label:"Delete",value:"/event_log/delete",pid:5},
                    {id:8,label:"Retry",value:"/event_log/retry",pid:5},
                ]
            },
            {id:9,label:"DLQ Log",value:"",pid:4,children: [
                    {id:10,label:"Edit",value:"",pid:9},
                    {id:11,label:"Delete",value:"",pid:9},
                    {id:12,label:"Retry",value:"",pid:9}
                ]
            },
            {id:13,label: "Workflow Log",value:"",pid:4,children: [
                    {id:14,label:"Edit",value:"",pid:13},
                    {id:15,label:"Delete",value:"",pid:13},
                    {id:16,label:"Retry",value:"",pid:13}
                ]
            }
        ]
    },
    {id:17,label:"Redis",value:"",pid:0,children: [
            {id:18,label:"Info",value:"/redis",pid:17},
            {id:19,label:"Command",value:"/redis/monitor",pid:17}
        ]
    },
    {id:20,label:"Setting",value:"",pid:0,children: [
            {id:21,label:"Operation Log",value:"/log/opt_log",pid:20,children: [
                    {id:30,label:"Delete",value:"/log/opt_log",pid:21}
                ]
            },
            {id:22,label:"User",value:"/user/list",pid:20,children: [
                    {id:23,label:"Add",value:"/user/add",pid:22},
                    {id:24,label:"Delete",value:"/user/del",pid:22},
                    {id:25,label:"Edit",value:"/user/edit",pid:22}
                ]
            },
            {id:26,label:"Role",value:"",pid:20,children: [
                    {id:27,label:"Add",value:"",pid:26},
                    {id:28,label:"Delete",value:"",pid:26},
                    {id:29,label:"Edit",value:"",pid:26}
                ]
            }
        ]
    }
];