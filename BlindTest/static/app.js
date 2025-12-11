let ws = null;
let currentRoomId = null;
let currentPlayerId = null;
let username = '';
let gameTimer = null;
let timerDuration = 30;
let audio = null;

const screens = {
    home: document.getElementById('home-screen'),
    config: document.getElementById('config-screen'),
    lobby: document.getElementById('lobby-screen'),
    game: document.getElementById('game-screen'),
    roundEnd: document.getElementById('round-end-screen'),
    end: document.getElementById('end-screen')
};

document.addEventListener('DOMContentLoaded', () => {
    setupEventListeners();
    audio = document.getElementById('audio-player');
});

function setupEventListeners() {
    document.getElementById('create-room-btn').addEventListener('click', showConfigScreen);
    document.getElementById('join-room-btn').addEventListener('click', joinRoom);
    document.getElementById('confirm-config-btn').addEventListener('click', createRoom);
    document.getElementById('cancel-config-btn').addEventListener('click', () => showScreen('home'));
    document.getElementById('ready-btn').addEventListener('click', setReady);
    document.getElementById('leave-room-btn').addEventListener('click', leaveRoom);
    document.getElementById('submit-answer-btn').addEventListener('click', submitAnswer);
    document.getElementById('playlist-select').addEventListener('change', updateGenreDescription);
    document.getElementById('answer-input').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            submitAnswer();
        }
    });
    document.getElementById('back-home-btn').addEventListener('click', () => {
        if (ws) {
            ws.close();
        }
        showScreen('home');
    });
}

function connectWebSocket() {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws`;
    
    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
        console.log('WebSocket connected');
    };

    ws.onmessage = (event) => {
        const message = JSON.parse(event.data);
        handleMessage(message);
    };

    ws.onerror = (error) => {
        console.error('WebSocket error:', error);
    };

    ws.onclose = () => {
        console.log('WebSocket disconnected');
    };
}

function handleMessage(message) {
    switch (message.type) {
        case 'room_created':
            currentRoomId = message.roomId;
            currentPlayerId = message.data.playerId;
            document.getElementById('room-code').textContent = currentRoomId;
            showScreen('lobby');
            break;

        case 'room_joined':
            currentRoomId = message.roomId;
            currentPlayerId = message.data.playerId;
            document.getElementById('room-code').textContent = currentRoomId;
            showScreen('lobby');
            break;

        case 'player_list':
            updatePlayerList(message.data.players);
            break;

        case 'game_start':
            document.getElementById('max-rounds').textContent = message.data.maxRounds;
            showScreen('game');
            break;

        case 'round_start':
            startRound(message.data);
            break;

        case 'round_end':
            endRound(message.data);
            break;

        case 'correct_answer':
            showCorrectNotification(message.data);
            break;

        case 'game_end':
            endGame(message.data);
            break;

        case 'error':
            alert(message.data.message);
            break;
    }
}

function showConfigScreen() {
    const usernameInput = document.getElementById('username-input');
    username = usernameInput.value.trim();

    if (!username) {
        alert('Veuillez entrer un pseudo');
        return;
    }

    document.getElementById('config-username').value = username;
    showScreen('config');
}

function createRoom() {
    const playlist = document.getElementById('playlist-select').value;
    const rounds = parseInt(document.getElementById('rounds-input').value);
    const time = parseInt(document.getElementById('time-input').value);

    if (rounds < 1 || rounds > 20) {
        alert('Le nombre de manches doit être entre 1 et 20');
        return;
    }

    if (time < 10 || time > 60) {
        alert('Le temps par manche doit être entre 10 et 60 secondes');
        return;
    }

    timerDuration = time;

    connectWebSocket();

    ws.onopen = () => {
        ws.send(JSON.stringify({
            type: 'create_room',
            username: username,
            playlist: playlist,
            maxRounds: rounds,
            roundTime: time
        }));
    };
}

function joinRoom() {
    const usernameInput = document.getElementById('username-input');
    const roomCodeInput = document.getElementById('room-code-input');
    username = usernameInput.value.trim();
    const roomCode = roomCodeInput.value.trim().toUpperCase();

    if (!username) {
        alert('Veuillez entrer un pseudo');
        return;
    }

    if (!roomCode) {
        alert('Veuillez entrer un code de room');
        return;
    }

    connectWebSocket();

    ws.onopen = () => {
        ws.send(JSON.stringify({
            type: 'join_room',
            username: username,
            roomId: roomCode
        }));
    };
}

function setReady() {
    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
            type: 'ready'
        }));
        document.getElementById('ready-btn').disabled = true;
        document.getElementById('ready-btn').textContent = 'En attente...';
    }
}

function leaveRoom() {
    if (ws) {
        ws.close();
    }
    currentRoomId = null;
    currentPlayerId = null;
    document.getElementById('ready-btn').disabled = false;
    document.getElementById('ready-btn').textContent = 'Prêt !';
    showScreen('home');
}

function submitAnswer() {
    const answerInput = document.getElementById('answer-input');
    const answer = answerInput.value.trim();

    if (!answer) {
        return;
    }

    if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
            type: 'answer',
            answer: answer
        }));
    }

    answerInput.value = '';
}

function updatePlayerList(players) {
    const containers = [
        document.getElementById('players-container'),
        document.getElementById('game-players-container'),
        document.getElementById('round-players-container')
    ];

    containers.forEach(container => {
        if (container) {
            container.innerHTML = '';
            players.forEach(player => {
                const playerDiv = document.createElement('div');
                playerDiv.className = 'player-item';
                
                const nameSpan = document.createElement('span');
                nameSpan.className = 'player-name';
                nameSpan.textContent = player.username;
                
                const scoreSpan = document.createElement('span');
                scoreSpan.className = 'player-score';
                scoreSpan.textContent = `${player.score} pts`;
                
                playerDiv.appendChild(nameSpan);
                
                if (player.ready && container === document.getElementById('players-container')) {
                    const readySpan = document.createElement('span');
                    readySpan.className = 'player-ready';
                    readySpan.textContent = '✓ Prêt';
                    playerDiv.appendChild(readySpan);
                } else {
                    playerDiv.appendChild(scoreSpan);
                }
                
                container.appendChild(playerDiv);
            });
        }
    });
}

function startRound(data) {
    if (screens.roundEnd.classList.contains('active')) {
        showScreen('game');
    }

    document.getElementById('current-round').textContent = data.round;
    document.getElementById('answer-input').value = '';
    document.getElementById('answer-input').disabled = false;
    document.getElementById('submit-answer-btn').disabled = false;

    if (audio) {
        audio.src = data.preview;
        audio.play().catch(err => console.error('Audio play error:', err));
    }

    const vinyl = document.getElementById('vinyl');
    vinyl.classList.add('spinning');

    startTimer();
}

function endRound(data) {
    if (gameTimer) {
        clearInterval(gameTimer);
    }

    const vinyl = document.getElementById('vinyl');
    vinyl.classList.remove('spinning');

    if (audio) {
        audio.pause();
    }

    document.getElementById('answer-input').disabled = true;
    document.getElementById('submit-answer-btn').disabled = true;

    document.getElementById('track-title').textContent = data.title;
    document.getElementById('track-artist').textContent = data.artist;
    document.getElementById('track-album').textContent = `Album: ${data.album}`;

    showScreen('roundEnd');
}

function endGame(data) {
    const resultsContainer = document.getElementById('results-container');
    resultsContainer.innerHTML = '';

    const sortedPlayers = data.players.sort((a, b) => b.score - a.score);

    sortedPlayers.forEach((player, index) => {
        const resultDiv = document.createElement('div');
        resultDiv.className = 'result-item';
        
        const rankSpan = document.createElement('span');
        rankSpan.className = 'result-rank';
        rankSpan.textContent = `#${index + 1}`;
        
        const nameSpan = document.createElement('span');
        nameSpan.className = 'result-name';
        nameSpan.textContent = player.username;
        
        const scoreSpan = document.createElement('span');
        scoreSpan.className = 'result-score';
        scoreSpan.textContent = `${player.score} pts`;
        
        resultDiv.appendChild(rankSpan);
        resultDiv.appendChild(nameSpan);
        resultDiv.appendChild(scoreSpan);
        
        resultsContainer.appendChild(resultDiv);

        setTimeout(() => {
            resultDiv.style.opacity = '0';
            resultDiv.style.transform = 'translateX(-20px)';
            setTimeout(() => {
                resultDiv.style.transition = 'all 0.5s ease';
                resultDiv.style.opacity = '1';
                resultDiv.style.transform = 'translateX(0)';
            }, 50);
        }, index * 200);
    });

    showScreen('end');
}

function showCorrectNotification(data) {
    const notification = document.getElementById('correct-notification');
    
    let message = '';
    let icon = '';
    
    switch(data.answerType) {
        case 'both':
            message = `${data.username} a trouvé titre + artiste! +${data.points} pts`;
            icon = '✅✅';
            break;
        case 'title_completing':
            message = `${data.username} a complété avec le titre! +${data.points} pts`;
            icon = '✅';
            break;
        case 'artist_completing':
            message = `${data.username} a complété avec l'artiste! +${data.points} pts`;
            icon = '✅';
            break;
        case 'title_partial':
            message = `${data.username} a trouvé le titre! +${data.points} pts (moitié)`;
            icon = '🟠';
            break;
        case 'artist_partial':
            message = `${data.username} a trouvé l'artiste! +${data.points} pts (moitié)`;
            icon = '🟠';
            break;
        default:
            message = `${data.username} +${data.points} pts`;
            icon = '✅';
    }
    
    notification.textContent = `${icon} ${message}`;
    notification.style.background = data.color === 'green' ? '#00b893db' : '#ff8c00e6';
    notification.classList.add('show');

    setTimeout(() => {
        notification.classList.remove('show');
    }, 3000);
}

function startTimer() {
    let elapsed = 0;
    const timerBar = document.getElementById('timer-bar');

    if (gameTimer) {
        clearInterval(gameTimer);
    }

    gameTimer = setInterval(() => {
        elapsed += 0.1;
        const percentage = 100 - (elapsed / timerDuration * 100);
        timerBar.style.width = percentage + '%';

        if (elapsed >= timerDuration) {
            clearInterval(gameTimer);
        }
    }, 100);
}

function showScreen(screenName) {
    Object.values(screens).forEach(screen => {
        screen.classList.remove('active');
    });

    if (screens[screenName]) {
        screens[screenName].classList.add('active');
    }
}

function updateGenreDescription() {
    const genre = document.getElementById('playlist-select').value;
    const descriptionElement = document.getElementById('genre-description');
    
    const descriptions = {
        'generale': 'Un mélange de tous les genres musicaux pour une expérience variée (Pop, Rock, Jazz, Metal...)',
        'francaise': 'Les plus grands hits de la musique française (Stromae, Angèle, Édith Piaf, Julien Doré...)',
        'pop': 'Les hits pop du moment et les classiques incontournables (Taylor Swift, Ed Sheeran, Dua Lipa, ABBA...)',
        'rock': 'Du rock classique au rock moderne, guitares électriques garanties (Queen, Nirvana, Foo Fighters, AC/DC...)',
        'rap': 'Hip-hop et rap, des pionniers aux nouveaux talents (Eminem, Tupac, BigFlo & Oli...)',
        'electronic': 'Musique électronique, dance et EDM pour faire vibrer la piste (Daft Punk, Calvin Harris, Avicii, David Guetta...)',
        'indie': 'Sons alternatifs et indépendants hors des sentiers battus (Arctic Monkeys, The Strokes, Tame Impala, Arcade Fire...)',
        'classic': 'Les grands classiques de la musique, intemporels (Beethoven, Mozart, Chopin, Vivaldi...)',
        'country': 'Country et folk américain, guitares acoustiques et banjos (Johnny Cash, Dolly Parton, Luke Combs...)',
        'jazz': 'Jazz classique et moderne, improvisations et swing (Miles Davis, Ella Fitzgerald, Louis Armstrong...)',
        'blues': 'Les racines du blues, guitares et harmonica (ZZ Top (blues rock), Muddy Waters, Eric Clapton...)',
        'reggae': 'Rythmes jamaïcains et vibes positives (Bob Marley, Peter Tosh, Damian Marley...)',
        'rnb': 'R&B moderne et contemporain, voix soul et beats smooth (The Weeknd, Beyoncé...)',
        'soul': 'Soul et funk, grooves irrésistibles et voix puissantes (Aretha Franklin, James Brown, Stevie Wonder...)',
        'metal': 'Metal dans tous ses styles, du heavy au progressive (Metallica, Iron Maiden, Slayer, System of a Down...)',
        'alternative': 'Rock alternatif et sons expérimentaux (Radiohead, Muse, The Killers, Linkin Park ...)',
        'latin': 'Musique latine, reggaeton, salsa et rythmes ensoleillés (Shakira, J Balvin ...)',
        'techno':'Techno underground et futuriste, rythmes répétitifs et atmosphère électronique (...)'
    };
    
    descriptionElement.textContent = descriptions[genre] || '';
}
