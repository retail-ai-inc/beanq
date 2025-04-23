const dashboardApi = {
    Graphic(){
        return request.get("dashboard");
    },
    Total(){
        return request.get("dashboard/total");
    },
    Pods(){
        return request.get("dashboard/pods");
    },
    Nodes(){
        return request.get("nodes");
    },
    QueueLine(queues,execTime,count){

        let subtextNotice = `${execTime}`;
        if(execTime > 60){
            execTime = Math.floor(execTime / 60);
            subtextNotice = `${execTime}m`;
        }

        let series = [
            {
                data: queues,
                type: 'scatter',
                symbolSize: function (data) {
                    let size = Math.sqrt(data[2]) / 2.5;
                    if (size > 15) {
                        size = 15;
                    }
                    if (size < 5) {
                        size = 5;
                    }
                    return size;
                },
                emphasis: {
                    focus: 'series',
                    label: {
                        show: false,
                        position: 'top'
                    }
                },
                itemStyle: {
                    shadowBlur: 10,
                    shadowColor: 'rgba(120, 36, 50, 0.5)',
                    shadowOffsetY: 5,
                    color: new echarts.graphic.RadialGradient(0.4, 0.3, 1, [
                        {
                            offset: 0,
                            color: 'rgb(251, 118, 123)'
                        },
                        {
                            offset: 1,
                            color: 'rgb(204, 46, 72)'
                        }
                    ])
                }
            },
        ];

        let lineOpt = {};

        lineOpt.title = {
            text: 'Queued messages',
            subtext:`(chart:last minute)(${subtextNotice})(Mouse scroll wheel to view more)`
        };
        lineOpt.tooltip = {
            trigger: 'item',
            formatter: function (param) {
                let html = `<div style="font-size: 12px">
                            ${param.data[3]}
                            <ul style="padding-left: .5rem;margin-bottom:0;list-style-type: none;">
                                <li><span style="color:#198754">Ready:</span>${param.data[0]}</li>
                                <li><span style="color:#dc3545">Pending:</span>${param.data[1]}</li>
                                <li><span style="color:#0d6efd">Total:</span>${param.data[2]}</li>
                            </ul>
                    </div>`;
                return html;
            }
        };
        lineOpt.legend = {
            top:'18%'
        };
        lineOpt.grid = {
            top:'30%',
            left: '3%',
            right: '4%',
            bottom: '3%',
            containLabel: true
        };
        lineOpt.xAxis = {
            splitLine: {
                lineStyle: {
                    type: 'dashed'
                }
            }
        };
        lineOpt.yAxis = {
            splitLine: {
                lineStyle: {
                    type: 'dashed'
                }
            },
            scale: true
        };
        lineOpt.dataZoom = [
            {
                type: 'inside',
                start: 0,
                end: 10
            },
        ];
        lineOpt.series = series;
        return lineOpt;
    },
    MessageRateLine(values,execTime){

        let xdata = [];
        let publish = [],confirm = [],deliver = [],redelivered = [],ack = [],get = [],nget = [];
        values.forEach((val,ind)=>{
            publish.push( parseInt( val[0] /10));
            nget.push(parseInt(val[1] / 10));
            xdata.push(val[3]);
        })
        confirm = deliver = redelivered = ack = get = publish;

        let subtextNotice = `${execTime}`;

        let line = {};
        line.title = {
            text: 'Message rates',
            subtext: `(chart:last minute)(${subtextNotice})`,
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
            data: xdata,
            axisLabel: {
                rotate: 45,
                fontSize: 12,
                inside: true
            }
        };
        line.yAxis = {
            type: 'value',
                axisLine: {
                show: true,
            },
            axisLabel: {
                formatter: function (value) {
                    if(value<1){
                        value = 0;
                    }
                    return value + '/s';
                }
            }
        };
        let seriesConfig = {
            symbol: 'none',
            sampling: 'lttb'
        };

        line.series = [
            {
                name: 'Publish',
                type: 'line',
                data: publish,
                ...seriesConfig
            },
            {
                name: 'Confirm',
                type: 'line',
                data: confirm,
                ...seriesConfig
            },
            {
                name: 'Deliver',
                type: 'line',
                data: deliver,
                ...seriesConfig
            },
            {
                name: 'Redelivered',
                type: 'line',
                data: redelivered,
                ...seriesConfig
            },
            {
                name: 'Acknowledge',
                type: 'line',
                data: ack,
                ...seriesConfig
            },
            {
                name: 'Get',
                type: 'line',
                data: get,
                ...seriesConfig
            },
            {
                name: 'Get(noack)',
                type: 'line',
                data: nget,
                ...seriesConfig
            }
        ];
        line.dataZoom = [
            {
                type: 'inside',
                start: 0,
                end: 10
            },
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