const dashboardApi = {
    Total(){
        return request.get("dashboard");
    },
    Nodes(){
        return request.get("nodes");
    },
    QueueLine(queues){

        let vals = Object.values(queues);
        let ready = [],unacked = [],total = [];

        vals.forEach(function (val,ind) {
            ready.push(val["ready"]);
            unacked.push(val["unacked"]);
            total.push(val["total"]);
        })

        let series = [
            {"name":"Ready","type":"line","data":ready},
            {"name":"Unacked","type":"line","data":unacked},
            {"name":"Total","type":"line","data":total}
        ];

        let lineOpt = {};
        lineOpt.title = {
            text: 'Queued messages',
            subtext: '(chart:last minute)(10s)'
        };
        lineOpt.tooltip = {
            trigger: 'axis'
        };
        lineOpt.legend = {
            data: ['Ready', 'Unacked', 'Total'],
            top:'18%'
        };
        lineOpt.grid = {
            top:'30%',
            left: '3%',
            right: '4%',
            bottom: '3%',
            containLabel: true
        };
        lineOpt.toolbox = {
            feature: {
                // saveAsImage: {}
            }
        };
        lineOpt.xAxis = {
            type: 'category',
            boundaryGap: false,
            data: Object.keys(queues),
        };
        lineOpt.yAxis = {
            type: 'value',
            axisLine: {
                show: true,
            }
        };
        lineOpt.series = series;
        return lineOpt;
    },
    MessageRateLine(values){


        let xdata = Object.keys(values);
        let ydata = Object.values(values);
        let publish = [],confirm = [],deliver = [],redelivered = [],ack = [],get = [],nget = [];
        ydata.forEach((val,ind)=>{
            publish.push( parseInt( val["ready"] /10));
            nget.push(parseInt(val["unacked"] / 10));
        })
        confirm = deliver = redelivered = ack = get = publish;

        let line = {};
        line.title = {
            text: 'Message rates',
            subtext: '(chart:last minute)(10s)',
        };
        line.tooltip = {
            trigger: 'axis'
        };
        line.legend={
            data: ['Publish', 'Confirm', 'Deliver', 'Redelivered', 'Acknowledge', 'Get(noack)'],
            top:'18%'
        };
        line.grid = {
            top:'30%',
            left: '3%',
            right: '4%',
            bottom: '3%',
            containLabel: true
        };
        line.toolbox = {
            feature: {
                // saveAsImage: {}
            }
        };
        line.xAxis = {
            type: 'category',
                boundaryGap: false,
                data: xdata
        };
        line.yAxis = {
            type: 'value',
                axisLine: {
                show: true,
            },
            axisLabel: {
                formatter: function (value) {
                    return value + '/s';
                }
            }
        };
        line.series = [
            {
                name: 'Publish',
                type: 'line',
                data: publish
            },
            {
                name: 'Confirm',
                type: 'line',
                data: confirm
            },
            {
                name: 'Deliver',
                type: 'line',
                data: deliver
            },
            {
                name: 'Redelivered',
                type: 'line',
                data: redelivered
            },
            {
                name: 'Acknowledge',
                type: 'line',
                data: ack
            },
            {
                name: 'Get',
                type: 'line',
                data: get
            },
            {
                name: 'Get(noack)',
                type: 'line',
                data: nget
            }
        ];
        return line;
    },
    BarOption(values){
        let bar = {};
        bar.title = {
            text: 'Queue Size',
            left: 'left'
        };
        bar.xAxis = {
            type: 'category',
                data: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']
        };
        bar.yAxis = {
            type: 'value'
        };
        bar.series = [
            {
                data: [120, 200, 350, 420, 170, 210, 130],
                type: 'bar',
                showBackground: true,
                backgroundStyle: {
                    color: 'rgba(180, 180, 180, 0.2)'
                }
            }
        ];
        return bar;
    },
    PieOption(values){
        let pie = {};
        pie.title = {
            text: 'Referer of a Website',
            subtext: 'Fake Data',
            left: 'center'
        };
        pie.tooltip = {
            trigger: 'item'
        };
        pie.legend = {
            orient: 'vertical',
                left: 'left'
        };
        pie.series = [
            {
                name: 'Access From',
                type: 'pie',
                radius: '50%',
                data: [
                    { value: 1048, name: 'Search Engine' },
                    { value: 735, name: 'Direct' },
                    { value: 580, name: 'Email' },
                    { value: 484, name: 'Union Ads' },
                    { value: 300, name: 'Video Ads' }
                ],
                emphasis: {
                    itemStyle: {
                        shadowBlur: 10,
                        shadowOffsetX: 0,
                        shadowColor: 'rgba(0, 0, 0, 0.5)'
                    }
                }
            }
        ]
        return pie
    }
}