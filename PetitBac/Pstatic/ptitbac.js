"use strict";

let socket = null;
let identifiantClient = null;
let pseudoAutomatique = "";
let pseudoEnvoyeAuto = false;
const urlParams = new URLSearchParams(window.location.search);
const body = document.body || document.getElementsByTagName("body")[0];
const salonCode = (urlParams.get("room") || (body ? body.getAttribute("data-room-code") : "") || "").trim().toUpperCase();
let validationZone = null;
let validationPlayer = null;
let validationCategory = null;
let validationAnswer = null;
let validationVotes = null;
let validationPending = null;
let validationInfo = null;
let btnValidateAccept = null;
let btnValidateReject = null;
let currentValidationId = null;

function renseignerPseudoAuto(pseudo) {
    if (!pseudo) {
        return;
    }
    pseudoAutomatique = pseudo;
    const input = document.getElementById("pseudo");
    if (input && !input.value) {
        input.value = pseudoAutomatique;
    }
    envoyerPseudoAuto();
}

function envoyerPseudoAuto() {
    if (!pseudoAutomatique || pseudoEnvoyeAuto || !socket || socket.readyState !== WebSocket.OPEN) {
        return;
    }
    socket.send(JSON.stringify({
        type: "join",
        name: pseudoAutomatique
    }));
    pseudoEnvoyeAuto = true;
}

function connecterWebSocket() {
    const proto = (window.location.protocol === "https:") ? "wss://" : "ws://";
    let url = proto + window.location.host + "/PetitBac/ws";
    if (salonCode) {
        url += "?room=" + encodeURIComponent(salonCode);
    }
    socket = new WebSocket(url);

    socket.onopen = function () {
        envoyerPseudoAuto();
    };

    socket.onmessage = function (event) {
        const data = JSON.parse(event.data);
        if (data.type === "state") {
            mettreAJourEtat(data);
            return;
        }
        if (data.type === "identity") {
            identifiantClient = data.id;
            envoyerPseudoAuto();
        }
    };

    socket.onclose = function () {
        console.log("WebSocket ferme");
    };

    socket.onerror = function (err) {
        console.error("WebSocket erreur :", err);
    };
}

function formatTemps(sec) {
    const m = Math.floor(sec / 60);
    const s = sec % 60;
    return (m < 10 ? "0" + m : m) + ":" + (s < 10 ? "0" + s : s);
}

function activerFormulaire(peutEnvoyer) {
    const inputs = document.querySelectorAll(".champ-nom, .champ-categorie, #btn-send");
    inputs.forEach(el => {
        el.disabled = !peutEnvoyer;
    });
}

function mettreAJourPreparation(etat, monJoueur) {
    const bloc = document.getElementById("zone-preparation");
    const info = document.getElementById("infos-preparation");
    const bouton = document.getElementById("btn-ready");
    if (!bloc || !info || !bouton) {
        return;
    }

    if (etat.gameOver) {
        bloc.style.display = "none";
        info.textContent = "Partie terminee.";
        bouton.disabled = true;
        return;
    }

    if (!etat.waitingRestart) {
        bloc.style.display = "none";
        bouton.disabled = true;
        return;
    }

    bloc.style.display = "block";
    const total = etat.readyTotal || 0;
    const seuil = total > 0 ? Math.floor(total / 3) + 1 : 1;
    let texte = `${etat.readyCount}/${etat.readyTotal} joueur(s) ont vote Oui.`;
    if (total > 0) {
        texte += ` Relance a partir de ${seuil} joueur(s) (> 1/3).`;
    }
    info.textContent = texte;
    bouton.disabled = !!(monJoueur && monJoueur.ready);
}

function mettreAJourValidation(etat, monJoueur) {
    if (!validationZone) {
        return;
    }
    if (!etat.validationActive || !etat.validationEntry) {
        validationZone.style.display = "none";
        currentValidationId = null;
        if (validationInfo) {
            validationInfo.textContent = "";
        }
        return;
    }

    const entry = etat.validationEntry;
    validationZone.style.display = "block";
    currentValidationId = entry.id;

    if (validationPlayer) {
        validationPlayer.textContent = entry.playerName || "Anonyme";
    }
    if (validationCategory) {
        validationCategory.textContent = entry.category || "";
    }
    if (validationAnswer) {
        validationAnswer.textContent = entry.answer || "";
    }
    if (validationVotes) {
        const votes = entry.votes || 0;
        const required = entry.required || 0;
        validationVotes.textContent = `${votes}/${required} validation(s) requises`;
    }
    if (validationPending) {
        const restants = Math.max(0, (etat.validationPending || 0) - 1);
        validationPending.textContent = restants > 0 ? `${restants} reponse(s) restantes apres celle-ci.` : "Derniere reponse a valider.";
    }

    const approvals = entry.approvals || {};
    const hasVoted = identifiantClient ? approvals[identifiantClient] : false;
    const targetId = entry.playerId;
    const peutVoter = Boolean(
        monJoueur &&
        monJoueur.active &&
        !etat.roundActive &&
        identifiantClient &&
        identifiantClient !== targetId &&
        !hasVoted &&
        !entry.completed
    );

    if (btnValidateAccept) {
        btnValidateAccept.disabled = !peutVoter;
    }
    if (btnValidateReject) {
        btnValidateReject.disabled = !peutVoter;
    }

    if (validationInfo) {
        if (peutVoter) {
            validationInfo.textContent = "Valide ou refuse la reponse proposee.";
        } else if (identifiantClient === targetId) {
            validationInfo.textContent = "Tu ne votes pas sur tes propres reponses.";
        } else if (!monJoueur || !monJoueur.active) {
            validationInfo.textContent = "Seuls les joueurs actifs de la manche votent.";
        } else if (hasVoted) {
            validationInfo.textContent = "Vote enregistre. En attente des autres joueurs.";
        } else if (entry.completed) {
            validationInfo.textContent = entry.accepted ? "Reponse acceptee." : "Reponse refusee.";
        } else {
            validationInfo.textContent = "En attente des autres joueurs.";
        }
    }
}

function envoyerValidationVote(approve) {
    if (!socket || socket.readyState !== WebSocket.OPEN || currentValidationId === null) {
        return;
    }
    socket.send(JSON.stringify({
        type: "validate",
        validationId: currentValidationId,
        approve: approve
    }));
}

function mettreAJourEtat(etat) {
    const lettreSpan = document.getElementById("lettre-affichee");
    if (lettreSpan) {
        lettreSpan.textContent = etat.letter;
    }

    const timerSpan = document.getElementById("timer-affiche");
    if (timerSpan) {
        timerSpan.textContent = formatTemps(etat.remainingSeconds);
    }

    const infoManches = document.getElementById("info-manches");
    if (infoManches) {
        if (etat.roundLimit) {
            const courante = Math.min(etat.roundNumber, etat.roundLimit);
            infoManches.textContent = `Manche ${courante}/${etat.roundLimit}`;
        } else {
            infoManches.textContent = `Manche ${etat.roundNumber}`;
        }
    }

    let monJoueur = null;
    if (identifiantClient) {
        monJoueur = etat.players.find(p => p.id === identifiantClient) || null;
    }

    const champTemps = document.getElementById("temps-manche");
    if (champTemps && etat.roundDuration) {
        champTemps.value = etat.roundDuration;
    }
    const champManches = document.getElementById("nb-manches");
    if (champManches && etat.roundLimit) {
        champManches.value = etat.roundLimit;
    }
    const champCategories = document.getElementById("categories-input");
    if (champCategories && Array.isArray(etat.categories)) {
        champCategories.value = etat.categories.join("\n");
    }

    const peutJouer = etat.roundActive && monJoueur ? monJoueur.active : etat.roundActive;
    activerFormulaire(peutJouer && !etat.gameOver);

    const msg = document.getElementById("message-manche");
    if (msg) {
        if (etat.gameOver) {
            msg.textContent = "Partie terminee : le nombre maximum de manches a ete atteint.";
        } else if (etat.roundActive) {
            if (monJoueur && !monJoueur.active) {
                msg.textContent = "Tu n'as pas confirme pour cette manche. Prepare-toi pour la prochaine.";
            } else {
                msg.textContent = "";
            }
        } else if (etat.waitingRestart) {
            msg.textContent = "Manche terminee. Vote \"Oui, je rejoue\" (plus d'un tiers requis).";
        } else {
            msg.textContent = "Manche terminee ! (un joueur a tout rempli ou le temps est ecoule)";
        }
    }

    mettreAJourPreparation(etat, monJoueur);
    mettreAJourValidation(etat, monJoueur);

    const tbody = document.getElementById("scores-body");
    if (!tbody) {
        return;
    }
    tbody.innerHTML = "";
    etat.players.forEach(function (p) {
        const tr = document.createElement("tr");

        const tdNom = document.createElement("td");
        tdNom.textContent = p.name || "Anonyme";

        const tdScoreManche = document.createElement("td");
        tdScoreManche.textContent = p.score + " pt" + (p.score > 1 ? "s" : "");

        const totalValue = p.totalScore || 0;
        const tdScoreTotal = document.createElement("td");
        tdScoreTotal.textContent = totalValue + " pt" + (totalValue > 1 ? "s" : "");

        const tdStatut = document.createElement("td");
        if (p.active) {
            tdStatut.textContent = "En jeu";
        } else if (p.ready) {
            tdStatut.textContent = "Pret";
        } else {
            tdStatut.textContent = "En attente";
        }

        tr.appendChild(tdNom);
        tr.appendChild(tdScoreManche);
        tr.appendChild(tdScoreTotal);
        tr.appendChild(tdStatut);
        tbody.appendChild(tr);
    });
}

async function loadUserInfo() {
    try {
        const response = await fetch("/api/user");
        if (!response.ok) {
            return;
        }
        const data = await response.json();

        if (data.authenticated) {
            const userDisplay = document.getElementById("userDisplay");
            if (userDisplay) {
                userDisplay.textContent = `Salut, ${data.pseudo} !!`;
            }
            renseignerPseudoAuto(data.pseudo);
        }
    } catch (error) {
        console.error("Erreur lors du chargement des informations utilisateur:", error);
    }
}

document.addEventListener("DOMContentLoaded", function () {
    validationZone = document.getElementById("validation-zone");
    validationPlayer = document.getElementById("validation-player");
    validationCategory = document.getElementById("validation-category");
    validationAnswer = document.getElementById("validation-answer");
    validationVotes = document.getElementById("validation-votes");
    validationPending = document.getElementById("validation-pending");
    validationInfo = document.getElementById("validation-info");
    btnValidateAccept = document.getElementById("btn-validate-accept");
    btnValidateReject = document.getElementById("btn-validate-reject");

    if (btnValidateAccept) {
        btnValidateAccept.addEventListener("click", function () {
            envoyerValidationVote(true);
        });
    }
    if (btnValidateReject) {
        btnValidateReject.addEventListener("click", function () {
            envoyerValidationVote(false);
        });
    }

    connecterWebSocket();
    loadUserInfo();

    const form = document.getElementById("form-joueur");
    if (form) {
        form.addEventListener("submit", function (e) {
            e.preventDefault();
            if (!socket || socket.readyState !== WebSocket.OPEN) {
                alert("Connexion WebSocket non disponible");
                return;
            }

            const pseudo = document.getElementById("pseudo").value.trim();
            if (pseudo !== "") {
                socket.send(JSON.stringify({
                    type: "join",
                    name: pseudo
                }));
                pseudoEnvoyeAuto = true;
            }

            const champs = document.querySelectorAll(".champ-categorie");
            const answers = {};
            champs.forEach(function (input) {
                const categorie = input.dataset.categorie;
                answers[categorie] = input.value;
            });

            socket.send(JSON.stringify({
                type: "answers",
                answers: answers
            }));
        });
    }

    const boutonPret = document.getElementById("btn-ready");
    if (boutonPret) {
        boutonPret.addEventListener("click", function () {
            if (!socket || socket.readyState !== WebSocket.OPEN) {
                return;
            }
            socket.send(JSON.stringify({type: "ready"}));
        });
    }

    const configForm = document.getElementById("config-form");
    if (configForm) {
        configForm.addEventListener("submit", function (e) {
            e.preventDefault();
            const message = document.getElementById("config-message");
            if (message) {
                message.textContent = "Envoi en cours...";
            }

            const temps = parseInt(document.getElementById("temps-manche").value, 10);
            const manches = parseInt(document.getElementById("nb-manches").value, 10);
            const categoriesBrutes = document.getElementById("categories-input").value.split(/\r?\n/);
            const categories = categoriesBrutes.map(c => c.trim()).filter(c => c.length > 0);

            let configURL = "/PetitBac/config";
            if (salonCode) {
                configURL += "?room=" + encodeURIComponent(salonCode);
            }

            fetch(configURL, {
                method: "POST",
                headers: {"Content-Type": "application/json"},
                body: JSON.stringify({temps: temps, manches: manches, categories: categories})
            }).then(resp => {
                if (!resp.ok) {
                    throw new Error("Erreur serveur");
                }
                return resp.json();
            }).then(() => {
                if (message) {
                    message.textContent = "Parametres mis a jour ! Nouvelle partie en preparation...";
                }
            }).catch(err => {
                console.error(err);
                if (message) {
                    message.textContent = "Impossible de mettre a jour la configuration.";
                }
            });
        });
    }
});
