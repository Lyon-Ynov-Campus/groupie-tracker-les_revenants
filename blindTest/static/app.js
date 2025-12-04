let ws = new WebSocket("ws://" + location.host + "/ws");
let playerName = prompt("Entrez votre pseudo :");
ws.onopen = () => {
    ws.send(JSON.stringify({ type: "join", player: playerName }));
};

ws.onmessage = (event) => {
    let data = JSON.parse(event.data);
    if (data.type === "song") {
        document.getElementById("player").src = data.file;
        document.getElementById("player").play();
    } else if (data.type === "scoreboard") {
        let sb = document.getElementById("scoreboard");
    }
}
