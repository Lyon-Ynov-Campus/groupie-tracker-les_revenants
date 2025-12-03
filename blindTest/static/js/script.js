(() => {
  const wsUrl = window.BLINDTEST_WS_URL;
  const ws = new WebSocket((location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + wsUrl);

  const audio = document.getElementById('audio');
  const timerEl = document.getElementById('timer');
  const feedback = document.getElementById('feedback');
  const answerInput = document.getElementById('answerInput');
  const sendBtn = document.getElementById('sendBtn');
  const scoresList = document.getElementById('scoresList');

  let roundDeadline = null;
  let timerInterval = null;

  ws.onmessage = (ev) => {
    const msg = JSON.parse(ev.data);
    switch (msg.type) {
      case 'round.start':
        startRound(msg.payload);
        break;
      case 'answer.update':
        renderAnswerFeedback(msg.payload);
        break;
      case 'score.update':
        renderScoreUpdate(msg.payload);
        break;
      case 'round.end':
        stopTimer();
        feedback.textContent = 'Manche terminée.';
        break;
      case 'scoreboard.show':
        renderFinalScoreboard(msg.payload);
        break;
    }
  };

  sendBtn.addEventListener('click', () => {
    const ans = (answerInput.value || '').trim();
    if (!ans) return;
    ws.send(JSON.stringify({
      type: 'answer.submit',
      payload: { answer: ans }
    }));
    answerInput.value = '';
  });

  function startRound(payload) {
    const preview = payload.previewUrl;
    const secs = payload.responseSec;
    feedback.textContent = '';
    if (preview) {
      audio.src = preview;
      audio.play().catch(() => { /* ignore autoplay block */ });
    } else {
      audio.removeAttribute('src');
    }
    roundDeadline = Date.now() + secs * 1000;
    startTimer();
  }

  function startTimer() {
    stopTimer();
    timerInterval = setInterval(() => {
      const remaining = Math.max(0, roundDeadline - Date.now());
      const s = Math.ceil(remaining / 1000);
      timerEl.textContent = formatSeconds(s);
      if (remaining <= 0) {
        stopTimer();
      }
    }, 250);
  }

  function stopTimer() {
    if (timerInterval) {
      clearInterval(timerInterval);
      timerInterval = null;
    }
  }

  function formatSeconds(s) {
    const mm = String(Math.floor(s / 60)).padStart(2, '0');
    const ss = String(s % 60).padStart(2, '0');
    return `${mm}:${ss}`;
  }

  function renderAnswerFeedback(payload) {
    const status = payload.status;
    feedback.textContent = status === 'correct' ? 'Bonne réponse !' : 'Incorrect.';
  }

  function renderScoreUpdate(payload) {
    const li = document.createElement('li');
    li.textContent = `Joueur ${payload.userID}: +${payload.points} (Total: ${payload.total})`;
    scoresList.appendChild(li);
  }

  function renderFinalScoreboard(payload) {
    scoresList.innerHTML = '';
    const scores = payload.scores || {};
    const entries = Object.entries(scores).sort((a,b) => b[1] - a[1]);
    for (const [uid, total] of entries) {
      const li = document.createElement('li');
      li.textContent = `Joueur ${uid}: ${total} pts`;
      scoresList.appendChild(li);
    }
    feedback.textContent = 'Partie terminée.';
  }
})();
