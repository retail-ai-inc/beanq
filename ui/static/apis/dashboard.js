const dashboardApi = {
    Total(){
        return request.get("dashboard")
    },
    QueueLine(queues){

        let vals = Object.values(queues);
        let ready = [];
        let unacked = [];
        let total = [];

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
            subtext: '(chart:last minute)(?)'
        };
        lineOpt.tooltip = {
            trigger: 'axis'
        };
        lineOpt.legend = {
            data: ['Ready', 'Unacked', 'Total']
        };
        lineOpt.grid = {
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

        let line = {};
        line.title = {
            text: 'Message rates',
            subtext: '(chart:last minute)(?)',
        };
        line.tooltip = {
            trigger: 'axis'
        };
        line.legend={
            data: ['Publish', 'Confirm', 'Deliver', 'Redelivered', 'Acknowledge', 'Get', 'Get(noack)']
        };
        line.grid = {
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
                data: ['09:02:10', '09:02:20', '09:02:30', '09:02:40', '09:02:50', '09:03:00']
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
                data: [120, 132, 101, 134, 90, 230]
            },
            {
                name: 'Confirm',
                type: 'line',
                data: [220, 182, 191, 234, 290, 330]
            },
            {
                name: 'Deliver',
                type: 'line',
                data: [150, 232, 201, 154, 190, 330]
            },
            {
                name: 'Redelivered',
                type: 'line',
                data: [320, 332, 301, 334, 390, 330]
            },
            {
                name: 'Acknowledge',
                type: 'line',
                data: [820, 932, 901, 934, 1290, 1330]
            },
            {
                name: 'Get',
                type: 'line',
                data: [820, 932, 901, 934, 1290, 1330]
            },
            {
                name: 'Get(noack)',
                type: 'line',
                data: [820, 932, 901, 934, 1290, 1330]
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