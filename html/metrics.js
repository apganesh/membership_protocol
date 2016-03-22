var evtServers = [];
var numEvents = 60;

var cpuData = [];
var memData = [];

function initializeEventSourceClient(addr, status, mainserver) {
    if (status === 3) {
        return
    }
    if (status === 2 && evtServers[addr] != undefined) {
        evtServers[addr].close()
        delete evtServers[addr]
        return
    } else if (evtServers[addr] === undefined || status === 4) {

    } else {
        return
    }


    //var source = new EventSource('https://' + addr + '/events');

    var x = addr.split(":")
    var source;
    if (x[1] === undefined) {
        console.log("Creating  EventSource Client for: " + addr)
        source = new EventSource('https://' + addr + '/events');
    } else {
        console.log("Creating event source " + ":" + x[1] + '/events')
        source = new EventSource('https://' + document.location.hostname + ':' + x[1] + '/events');
    }
    evtServers[addr] = source;

    source.onopen = function(event) {
        console.log("EventSource open for: " + addr);
    };
    source.addEventListener('error', function(e) {
        if (e.readyState == EventSource.CLOSED) {
            console.log("EventSource closed for:" + addr)
        }
    }, false);

    if (mainserver === 1) {
        source.addEventListener('status', function(e) {
            var data = JSON.parse(e.data);
            updateEventLogs(data)
            updateMemberStatus(data)
        }, false);
    }

    source.addEventListener('metrics', function(e) {
        var data = JSON.parse(e.data);
        updateMetricsData(data)
    }, false);
}

function initializeMetricsData(d) {
    if (cpuData[d.IPAddress] === undefined || d.Status == 4) {

        console.log("Creating the cpuData for " + d.IPAddress)
        cpuData[d.IPAddress] = d3.range(numEvents).map(function() {
            return 50;
        });
        memData[d.IPAddress] = d3.range(numEvents).map(function() {
            return 50;
        });

    }
}

// Create the default event source ...
initializeEventSourceClient(document.location.host, 1, 1)

var textarea = document.getElementById('eventlog');
textarea.scrollTop = textarea.scrollHeight;

function updateEventLogs(data) {
    var logobj = document.getElementById("eventlog");
    for (var i = 0; i < data.length; i++) {
        var log = "\n" + data[i].Timestamp + " : " + data[i].IPAddress + " : " + statusString[data[i].Status]
        var txt = document.createTextNode(log)
        logobj.appendChild(txt)
    }
    logobj.scrollTop = logobj.scrollHeight;

}


function randomIntFromInterval(min, max) {
    return Math.floor(Math.random() * (max - min + 1) + min);
}

function updateMetricsData(newdata) {
    // We always get only one event .... 
    // Guaranteed its only one element
    var ip = newdata[0].IPAddress
    var cpuload = newdata[0].CPULoad
    var memusage = newdata[0].MemUsage

    var memberRow = d3.selectAll(".memberRow").filter(function(d) {
        var elt = d3.select(this)
        return elt.attr("ip") === ip
    });

    var cpuline = memberRow.selectAll("path").filter(function(d) {
        var elt = d3.select(this)
        return elt.attr("class") === "cpuline"
    })

    var memline = memberRow.selectAll("path").filter(function(d) {
        var elt = d3.select(this)
        return elt.attr("class") === "memline"
    })

    cpuData[ip].push(+cpuload);
    memData[ip].push(+memusage);

    cpuline.attr("d", valueline)
        .attr("transform", null)
        .transition()
        .duration(800)
        .ease("linear")
        .attr("transform", "translate(" + xD(-1) + ",0)");

    memline.attr("d", valueline)
        .attr("transform", null)
        .transition()
        .duration(800)
        .ease("linear")
        .attr("transform", "translate(" + xD(-1) + ",0)");

    cpuData[ip].shift();
    memData[ip].shift();
}
