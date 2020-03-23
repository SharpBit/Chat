const ws = new WebSocket("ws://localhost:8080/ws");

ws.onopen = function() {
    $("#status").html("Status: Connected");

    // Send a heartbeat every 25 seconds
    setInterval(() => {
        try {
            ws.send(JSON.stringify({ action: "HEARTBEAT" }));
        } catch(err) {
            console.error(err);
        }
    }, 25000);
}

ws.onerror = function(err) {
    console.error(err);
    $("#status").html("Status: Error");
}

ws.onclose = function(event) {
    $("#status").html("Status: Closed, Code: " + event.code + " " + event.reason);
}

ws.onmessage = function(event) {
    const d = JSON.parse(event.data);
    if(d && d.action === "MESSAGE_CREATE") {
        const control = $('#log');
        control.html(control.html() + `<b>${preventXSS(d.data.author)}</b> ${preventXSS(d.data.content)}<br />`);
        control.scrollTop(control.scrollTop() + 1000);
    }
}
