"use strict";

let waitingSocket = null;
let waitingRoomCode = "";
let waitingPseudo = "";
let pseudoEnvoye = false;

function getRoomCode() {
    if (waitingRoomCode) {
        return waitingRoomCode;
    }
    const body = document.body || document.getElementsByTagName("body")[0];
    if (body && body.dataset.roomCode) {
        waitingRoomCode = body.dataset.roomCode.trim().toUpperCase();
    } else {
        const params = new URLSearchParams(window.location.search);
        waitingRoomCode = (params.get("room") || "").trim().toUpperCase();
    }
    return waitingRoomCode;
}

async function fetchUserPseudo() {
    try {
        const response = await fetch("/api/user");
        if (!response.ok) {
            return "";
        }
        const data = await response.json();
        waitingPseudo = data && data.pseudo ? data.pseudo : "";
        envoyerPseudoSalle();
    } catch (err) {
        console.error("PetitBac: impossible de recuperer l'utilisateur", err);
    }
    return waitingPseudo;
}

function connecterSalle() {
    const code = getRoomCode();
    if (!code) {
        console.error("PetitBac: aucun code de salle pour la salle d'attente");
        return;
    }
    const proto = (window.location.protocol === "https:") ? "wss://" : "ws://";
    const url = proto + window.location.host + "/ws?room=" + encodeURIComponent(code);
    waitingSocket = new WebSocket(url);

    waitingSocket.onopen = function () {
        envoyerPseudoSalle();
    };

    waitingSocket.onmessage = function (event) {
        const data = JSON.parse(event.data);
        if (data.type === "state") {
            rendreEtatAttente(data);
            if (data.roundActive) {
                window.location.href = "/PetitBac/play?room=" + encodeURIComponent(code);
            }
        }
    };

    waitingSocket.onclose = function () {
        console.log("PetitBac: socket attente fermee");
    };

    waitingSocket.onerror = function (err) {
        console.error("PetitBac: erreur socket attente", err);
    };
}

function envoyerPseudoSalle() {
    if (!waitingPseudo || pseudoEnvoye || !waitingSocket || waitingSocket.readyState !== WebSocket.OPEN) {
        return;
    }
    waitingSocket.send(JSON.stringify({type: "join", name: waitingPseudo}));
    pseudoEnvoye = true;
}

function rendreEtatAttente(state) {
    const tbody = document.getElementById("waiting-scores");
    if (tbody) {
        tbody.innerHTML = "";
        state.players.forEach(function (player) {
            const tr = document.createElement("tr");
            const tdName = document.createElement("td");
            tdName.textContent = player.name || "Anonyme";
            const tdScore = document.createElement("td");
            const total = player.totalScore || 0;
            tdScore.textContent = total + " pt" + (total > 1 ? "s" : "");
            const tdStatus = document.createElement("td");
            tdStatus.textContent = player.active ? "En jeu" : (player.ready ? "Pret" : "En attente");
            tr.appendChild(tdName);
            tr.appendChild(tdScore);
            tr.appendChild(tdStatus);
            tbody.appendChild(tr);
        });
    }
    const info = document.getElementById("waiting-status");
    if (info) {
        info.textContent = `${state.players.length} joueur(s) connecte(s)`;
    }
}

function setupStartButton() {
    const button = document.getElementById("btn-start");
    if (!button) {
        return;
    }
    button.addEventListener("click", async function () {
        const code = getRoomCode();
        if (!code) {
            return;
        }
        const status = document.getElementById("waiting-status");
        if (status) {
            status.textContent = "Lancement en cours...";
        }
        const pseudo = waitingPseudo || await fetchUserPseudo();
        try {
            const response = await fetch("/PetitBac/rooms/start", {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify({code: code, host: pseudo})
            });
            if (!response.ok) {
                throw new Error("start failed");
            }
            if (status) {
                status.textContent = "Partie en cours de lancement...";
            }
        } catch (err) {
            console.error(err);
            if (status) {
                status.textContent = "Impossible de lancer la partie.";
            }
        }
    });
}

document.addEventListener("DOMContentLoaded", function () {
    getRoomCode();
    fetchUserPseudo();
    connecterSalle();
    setupStartButton();
});
