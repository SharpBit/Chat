$("#message").on("keydown", (event) => {
    const code = event.keyCode ? event.keyCode : event.which;

    // Enter (not shift+enter) can send messages
    if (code === 13 && !event.shiftKey) {
        $("#send").trigger("click");
    }
});

$("#send").on("click", (event) => {
    const msg = $("#message").val();
    if (!msg) return
    if (msg.length > 2000) return alert("Message must be less than 2000 characters.");
    try {
        try {
            ws.send(JSON.stringify({
                action: "MESSAGE_CREATE",
                data: {
                    content: msg,
                    author: $("#name").val() || "Anonymous"
              }
            }));
        } catch(error) {
        console.error(error);
        }

        // Clear the Message Box
        $("#message").val("").focus();
    } catch(err) {
        alert("ERROR: " + err);
    }
});