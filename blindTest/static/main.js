let currentRoomCode = null;
let currentPseudonym = null;
let gameSocket = null;
let gameData = null;

function showScreen(screenId) {
    document.querySelectorAll('.screen').forEach(s => s.classList.remove('active'));
    document.getElementById(screenId).classList.add('active');
}

function goHome() {
    showScreen('home-screen');
    if (gameSocket) {
        gameSocket.close();
        gameSocket = null;
    }
}

function showCreateGame() {
    showScreen('create-game-screen');
}

function showJoinGame() {
    showScreen('join-game-screen');
}

function createGame() {
    const playlist = document.getElementById('playlist').value;
    const maxRounds = parseInt(document.getElementById('max-rounds').value);
    const roundDuration = parseInt(document.getElementById('round-duration').value);
    const pseudonym = document.getElementById('pseudonym').value;

    if (!pseudonym.trim()) {
        alert('Veuillez entrer votre pseudonyme');
        return;
    }

    console.log('Création de partie avec:', { playlist, maxRounds, roundDuration, pseudonym });

    fetch('/api/create-game', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ playlist, maxRounds, roundDuration })
    })
    .then(res => {
        console.log('Response status:', res.status);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);
        return res.json();
    })
    .then(data => {
        console.log('Game created:', data);
        currentRoomCode = data.roomCode;
        currentPseudonym = pseudonym;
        
        return fetch(`/api/join-game?code=${currentRoomCode}&pseudonym=${pseudonym}`);
    })
    .then(res => {
        console.log('Join response:', res.status);
        if (!res.ok) throw new Error(`Join failed: ${res.status}`);
        return res.json();
    })
    .then(() => {
        console.log('Entering game...');
        enterGame();
    })
    .catch(err => {
        console.error('Error:', err);
        alert('Erreur: ' + err.message);
    });
}

function joinGame() {
    const roomCode = document.getElementById('room-code').value;
    const pseudonym = document.getElementById('join-pseudonym').value;

    if (!roomCode.trim() || !pseudonym.trim()) {
        alert('Veuillez remplir tous les champs');
        return;
    }

    fetch(`/api/join-game?code=${roomCode}&pseudonym=${pseudonym}`)
        .then(res => {
            if (!res.ok) throw new Error('Code invalide');
            return res.json();
        })
        .then(() => {
            currentRoomCode = roomCode;
            currentPseudonym = pseudonym;
            enterGame();
        })
        .catch(err => {
            alert('Erreur: ' + err.message);
        });
}

function enterGame() {
    showScreen('game-screen');
    
    initWebSocket();
    
    fetch(`/api/game-status?code=${currentRoomCode}`)
        .then(res => res.json())
        .then(data => {
            console.log('Game status data:', data);
            
            document.getElementById('round-display').textContent = `0/${data.MaxRounds}`;
            
            const playlistFlags = {
                'Rock': '🎸',
                'Rap': '🎤',
                'Pop': '🎵'
            };
            document.getElementById('playlist-flag').textContent = playlistFlags[data.Playlist] || '🎵';
            document.getElementById('playlist-name').textContent = data.Playlist;
            
            const playersDiv = document.getElementById('players');
            playersDiv.innerHTML = '';
            
            if (data && data.Players && Object.keys(data.Players).length > 0) {
                const players = Object.values(data.Players);
                players.sort((a, b) => b.Score - a.Score);
                
                players.forEach((player, index) => {
                    const playerItem = document.createElement('div');
                    playerItem.className = 'player-rank-item';
                    playerItem.innerHTML = `
                        <div class="rank-number">#${index + 1}</div>
                        <div class="rank-info">
                            <div class="rank-name">${player.Pseudonym}</div>
                            <div class="rank-score">${player.Score} pts</div>
                        </div>
                    `;
                    playersDiv.appendChild(playerItem);
                });
            }
            
            if (!data.Started) {
                document.getElementById('start-btn').style.display = 'block';
                document.getElementById('waiting-screen').style.display = 'block';
                document.getElementById('answer-form').style.display = 'none';
            }
        })
        .catch(err => {
            console.error('Error fetching game status:', err);
        });
}

function initWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const url = `${protocol}//${window.location.host}/ws/game?code=${currentRoomCode}&pseudonym=${currentPseudonym}`;
    
    gameSocket = new WebSocket(url);
    
    gameSocket.onmessage = (event) => {
        const data = JSON.parse(event.data);
        console.log('WebSocket message:', data);
        
        if (data.type === 'round_start') {
            handleRoundStart(data);
        } else if (data.type === 'round_end') {
            handleRoundEnd(data);
        } else if (data.type === 'game_end') {
            showScoreboard(data);
        } else {
            updateGameDisplay(data);
        }
    };

    gameSocket.onerror = (error) => {
        console.error('WebSocket error:', error);
    };
    
    gameSocket.onclose = () => {
        console.log('WebSocket closed');
    };
}

function handleRoundStart(data) {
    console.log('Round start:', data);
    
    document.getElementById('round-display').textContent = `${data.roundNumber}/${data.maxRounds}`;
    
    const playersDiv = document.getElementById('players');
    playersDiv.innerHTML = '';
    
    const players = Object.values(data.players);
    players.sort((a, b) => b.Score - a.Score);
    
    players.forEach((player, index) => {
        const playerItem = document.createElement('div');
        playerItem.className = 'player-rank-item';
        playerItem.innerHTML = `
            <div class="rank-number">#${index + 1}</div>
            <div class="rank-info">
                <div class="rank-name">${player.Pseudonym}</div>
                <div class="rank-score">${player.Score} pts</div>
            </div>
        `;
        playersDiv.appendChild(playerItem);
    });
    
    document.getElementById('start-btn').style.display = 'none';
    document.getElementById('waiting-screen').style.display = 'none';
    document.getElementById('answer-form').style.display = 'flex';
    document.getElementById('responses-list').innerHTML = '';
    
    if (data.currentTrack) {
        if (data.currentTrack.previewUrl) {
            playMusic(data.currentTrack.previewUrl);
        }
    }
    
    document.getElementById('answer-input').value = '';
    document.getElementById('answer-input').focus();
}

function playMusic(previewUrl) {
    const audioPlayer = document.getElementById('audio-player');
    if (!audioPlayer) {
        const audio = document.createElement('audio');
        audio.id = 'audio-player';
        audio.style.display = 'none';
        document.body.appendChild(audio);
    }
    
    const audio = document.getElementById('audio-player');
    audio.src = previewUrl;
    audio.play().catch(err => console.log('Audio playback failed:', err));
}

function handleRoundEnd(data) {
    const audio = document.getElementById('audio-player');
    if (audio) {
        audio.pause();
        audio.currentTime = 0;
    }
    
    const playersDiv = document.getElementById('players');
    playersDiv.innerHTML = '';
    for (const [name, player] of Object.entries(data.players)) {
        const playerCard = document.createElement('div');
        playerCard.className = 'player-card';
        playerCard.innerHTML = `
            <div class="player-name">${name}</div>
            <div class="player-score">${player.Score}</div>
        `;
        playersDiv.appendChild(playerCard);
    }
    
    document.getElementById('answer-form').style.display = 'none';
    document.getElementById('waiting-screen').style.display = 'block';
    
    if (data.roundNumber < data.maxRounds) {
        document.getElementById('waiting-text').textContent = 'Prochaine manche...';
    } else {
        document.getElementById('waiting-text').textContent = 'Fin du jeu...';
    }
}

function updateGameStatus() {
    fetch(`/api/game-status?code=${currentRoomCode}`)
        .then(res => res.json())
        .then(data => {
            gameData = data;
            updateGameDisplay(data);
        })
        .catch(err => console.error(err));
}

function updateGameDisplay(data) {
    if (!data) return;

    document.getElementById('round-display').textContent = `${data.RoundNumber}/${data.MaxRounds}`;

    const playersDiv = document.getElementById('players');
    playersDiv.innerHTML = '';
    
    for (const [name, player] of Object.entries(data.Players)) {
        const playerCard = document.createElement('div');
        playerCard.className = 'player-card';
        playerCard.innerHTML = `
            <div class="player-name">${name}</div>
            <div class="player-score">${player.Score}</div>
        `;
        playersDiv.appendChild(playerCard);
    }

    if (!data.Started) {
        document.getElementById('start-btn').style.display = 'block';
        document.getElementById('waiting-screen').style.display = 'block';
        document.getElementById('answer-form').style.display = 'none';
        document.getElementById('waiting-text').textContent = 'En attente du début de la partie...';
    } else if (data.Finished) {
        showScoreboard(data);
    } else {
        document.getElementById('start-btn').style.display = 'none';
        document.getElementById('waiting-screen').style.display = 'none';
        document.getElementById('answer-form').style.display = 'flex';

        if (data.CurrentTrack) {
            document.getElementById('current-artist').textContent = data.CurrentTrack.Artist;
        }
    }
}

function startGame() {
    if (gameSocket && gameSocket.readyState === WebSocket.OPEN) {
        gameSocket.send(JSON.stringify({ type: 'start' }));
    }
}

function submitAnswer() {
    const answer = document.getElementById('answer-input').value;
    
    if (!answer.trim()) {
        return;
    }

    if (gameSocket && gameSocket.readyState === WebSocket.OPEN) {
        gameSocket.send(JSON.stringify({ type: 'answer', answer }));
        document.getElementById('answer-form').style.display = 'none';
        document.getElementById('waiting-screen').style.display = 'block';
        document.getElementById('waiting-text').textContent = 'Réponse envoyée...';
    }
}

function addFeedbackDot(status) {
    const container = document.getElementById('feedback-container');
    const dot = document.createElement('div');
    dot.className = `feedback-dot ${status}`;
    container.appendChild(dot);
}

function toggleVolume() {
    const audio = document.getElementById('audio-player');
    if (audio) {
        audio.muted = !audio.muted;
    }
}

function showScoreboard(data) {
    showScreen('scoreboard-screen');
    
    const scoreboard = document.getElementById('scoreboard');
    scoreboard.innerHTML = '';

    const players = data.players ? Object.values(data.players) : Object.values(data.Players || {});
    players.sort((a, b) => b.Score - a.Score);

    if (players.length === 0) {
        scoreboard.innerHTML = '<p>Aucun joueur</p>';
        return;
    }

    players.forEach((player, index) => {
        const row = document.createElement('div');
        row.className = 'scoreboard-row';
        row.innerHTML = `
            <span class="scoreboard-rank">#${index + 1}</span>
            <span class="scoreboard-name">${player.Pseudonym}</span>
            <span class="scoreboard-score">${player.Score} pts</span>
        `;
        scoreboard.appendChild(row);
    });
}

function leaveGame() {
    if (confirm('Êtes-vous sûr de vouloir quitter?')) {
        goHome();
    }
}

showScreen('home-screen');
