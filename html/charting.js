var xD = d3.scale.linear()
    .domain([0, 60])
    .range([0, 300]);

var yD = d3.scale.linear()
    .domain([0, 100])
    .range([100, 0]);

var valueline = d3.svg.line()
    .interpolate("cardinal")
    .x(function(d, i) {
        return xD(i);
    })
    .y(function(d) {
        return yD(d);
    });


var statusColor = {
    1: 'green',
    2: 'orange',
    3: 'red',
    4: 'green'
}

var statusString = {
    1: 'STARTED ',
    2: 'FAILING ',
    3: 'FAILED ',
    4: 'REJOINED '
}

function updateMemberStatus(newdata) {

    /////////
    //ENTER//
    /////////

    //Bind new data to chart rows 
    var memberBox = d3.select('#members')
    var memberRow = memberBox.selectAll("g.memberRow")

    //Create row for a new Node added
    var newRow = memberRow.data(
            newdata,
            function(d) {
                return d.IPAddress
            }
        ).enter()
        .append("g")
        .attr("class", "memberRow")
        .attr("ip", function(d) {
            initializeMetricsData(d);
            initializeEventSourceClient(d.IPAddress, d.Status, 0)
            return d.IPAddress;
        }).append("svg")
        .attr("width", "1000")
        .attr("height", "125");


    var rowElements = $(".memberRow");
    var rowCount = rowElements.length;

    //var xcoeff = rowCount % 3
    //var ycoeff = Math.floor(rowCount / 3)


    // Enclosing rectangle for Node information
    newRow.append("rect")
        .attr("class", "addrbar")
        .attr("y", 5)
        .attr("height", 120)
        .attr("width", 300)

    //Adding Node ipaddr information
    newRow.append("text")
        .attr("class", "nodelabel")
        .attr("y", 50)
        .attr("x", 150)
        .attr("width", 300)
        .text(function(d) {
            return d.IPAddress;
        })
        .attr("fill", function(d) {
            return statusColor[d.Status]
        });

    //Add Status and Time of the event
    newRow.append("text")
        .attr("class", "nodestatus")
        .attr("y", 100)
        .attr("x", 150)
        .text(function(d) {
            return statusString[d.Status] + ":" + d.Timestamp
        });


    // From bostock bl.ocks.org/mbostock/1642874

    var cpuChart = newRow.append("g");

    cpuChart.append("defs").append("clipPath")
        .attr("id", "clip")
        .append("rect")
        .attr("width", 300)
        .attr("height", 125)
        .attr("transform", "translate(0," + "0" + ")");

    // Xaxis
    cpuChart.append("g")
        .attr("class", "cpu xaxis")
        .attr("transform", "translate(0," + "100" + ")")
        .call(d3.svg.axis().scale(xD).orient("bottom").ticks(3));

    // Yaxis
    cpuChart.append("g")
        .attr("class", "cpu yaxis")
        .call(d3.svg.axis().scale(yD).orient("left").ticks(4));

    // Title
    cpuChart.append("text")
        .attr("class", "title")
        .attr("x", 300 / 2)
        .attr("y", 5)
        .attr("text-anchor", "middle")
        .text("CPU Load");

    // path line
    cpuChart
        .attr("class", "cpupath")
        .attr("transform", "translate(330," + "5" + ")")
        .append("path")
        .datum(function(d) {
            return cpuData[d.IPAddress]
        })
        .attr("class", "cpuline")
        .attr("d", valueline)


    var memChart = newRow.append("g");
    memChart.append("defs").append("clipPath")
        .attr("id", "clip")
        .append("rect")
        .attr("width", 300)
        .attr("height", 125);

    //XAxis
    memChart.append("g")
        .attr("class", "mem xaxis")
        .attr("transform", "translate(0," + "100" + ")")
        .call(d3.svg.axis().scale(xD).orient("bottom").ticks(3));
    // YAxis
    memChart.append("g")
        .attr("class", "mem yaxis")
        .call(d3.svg.axis().scale(yD).orient("left").ticks(4));

    // path line
    memChart
        .attr("class", "mempath")
        .attr("transform", "translate(660," + "5" + ")")
        .append("path")
        .datum(function(d) {
            return memData[d.IPAddress]
        })
        .attr("class", "memline")
        .attr("d", valueline);

    // Title
    memChart.append("text")
        .attr("class", "title")
        .attr("x", 300 / 2)
        .attr("y", 5)
        .attr("text-anchor", "middle")
        .text("Memory Usage");


    //////////
    //UPDATE//
    //////////


    //Update Node ipaddress
    memberRow.select(".nodelabel").transition()
        .duration(500)
        .attr("opacity", 1)
        .attr("fill", function(d) {
            //initializeMetricsData(d)
            return statusColor[d.Status]
        })
        .tween("text", function(d) {
            this.textContent = d.IPAddress
        });

    //
    memberRow.select(".nodestatus").transition()
        .duration(500)
        .attr("opacity", 1)
        .tween("text", function(d) {
            this.textContent = statusString[d.Status] + ": " + d.Timestamp;
        });


    ////////
    //EXIT//  WE DONT NEED AS WE ARE NOT REMOVING ALREADY EXISTING ROWS
    ///////

    // We need to transform only if a new machine is added
    if (newdata.length > 1 || newdata[0].Status == 1) {
        memberRow
            .transition()
            .delay(300)
            .duration(600)
            .attr("transform", function(d, i) {
                return "translate(" + 0 * 330 + "," + (rowCount + i) * 125 + ")";
            });
    }

};
