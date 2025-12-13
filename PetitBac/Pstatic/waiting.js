"use strict";

let waitingSocket = null;
let waitingRoomCode = "";
let waitingPseudo = "";
<<<<<<< HEAD
let pseudoSent = false;
=======
let pseudoEnvoye = false;
>>>>>>> v1seb

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
<<<<<<< HEAD
        trySendJoin();
    } catch (err) {
        console.error("PetitBac: utilisateur indisponible", err);
=======
        envoyerPseudoSalle();
    } catch (err) {
        console.error("PetitBac: impossible de recuperer l'utilisateur", err);
>>>>>>> v1seb
    }
    return waitingPseudo;
}

<<<<<<< HEAD
function connectWaitingSocket() {
    const code = getRoomCode();
    if (!code) {
        console.error("PetitBac: aucun code fourni pour la salle d'attente");
=======
function connecterSalle() {
    const code = getRoomCode();
    if (!code) {
        console.error("PetitBac: aucun code de salle pour la salle d'attente");
>>>>>>> v1seb
        return;
    }
    const proto = (window.location.protocol === "https:") ? "wss://" : "ws://";
    const url = proto + window.location.host + "/ws?room=" + encodeURIComponent(code);
    waitingSocket = new WebSocket(url);

    waitingSocket.onopen = function () {
<<<<<<< HEAD
        trySendJoin();
=======
        envoyerPseudoSalle();
>>>>>>> v1seb
    };

    waitingSocket.onmessage = function (event) {
        const data = JSON.parse(event.data);
        if (data.type === "state") {
<<<<<<< HEAD
            renderWaitingState(data);
=======
            rendreEtatAttente(data);
>>>>>>> v1seb
            if (data.roundActive) {
                window.location.href = "/PetitBac/play?room=" + encodeURIComponent(code);
            }
        }
    };

    waitingSocket.onclose = function () {
<<<<<<< HEAD
        console.log("PetitBac: socket salle d'attente ferme");
    };

    waitingSocket.onerror = function (err) {
        console.error("PetitBac: socket salle d'attente erreur", err);
    };
}

function trySendJoin() {
    if (!waitingPseudo || !waitingSocket || waitingSocket.readyState !== WebSocket.OPEN || pseudoSent) {
        return;
    }
    waitingSocket.send(JSON.stringify({type: "join", name: waitingPseudo}));
    pseudoSent = true;
}

function renderWaitingState(state) {
=======
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
>>>>>>> v1seb
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
<<<<<<< HEAD
            tdStatus.textContent = player.active ? "En jeu" : (player.ready ? "Prêt" : "En attente");
=======
            tdStatus.textContent = player.active ? "En jeu" : (player.ready ? "Pret" : "En attente");
>>>>>>> v1seb
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

<<<<<<< HEAD
async function loadPersistedScores() {
    const code = getRoomCode();
    if (!code) {
        return;
    }
    try {
        const response = await fetch(`/PetitBac/rooms/players?room=${encodeURIComponent(code)}`);
        if (!response.ok) {
            return;
        }
        const data = await response.json();
        if (!data || !Array.isArray(data.players)) {
            return;
        }
        const tbody = document.getElementById("waiting-scores");
        if (!tbody) {
            return;
        }
        tbody.innerHTML = "";
        data.players.forEach(function (player) {
            const tr = document.createElement("tr");
            const tdName = document.createElement("td");
            tdName.textContent = player.pseudo;
            const tdScore = document.createElement("td");
            tdScore.textContent = player.score + " pts";
            const tdStatus = document.createElement("td");
            tdStatus.textContent = "En attente";
            tr.appendChild(tdName);
            tr.appendChild(tdScore);
            tr.appendChild(tdStatus);
            tbody.appendChild(tr);
        });
    } catch (err) {
        console.warn("PetitBac: scores persistants indisponibles", err);
    }
}

=======
>>>>>>> v1seb
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
<<<<<<< HEAD
    loadPersistedScores();
    connectWaitingSocket();
=======
    connecterSalle();
>>>>>>> v1seb
    setupStartButton();
});
