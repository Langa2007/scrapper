(function () {
  'use strict';

  const script = document.currentScript || document.querySelector('script[data-api]');
  const API = (script && script.getAttribute('data-api')) || 'http://localhost:8080';
  const TITLE = (script && script.getAttribute('data-title')) || 'Assistant';
  const MODE = (script && script.getAttribute('data-mode')) || 'general';
  const CTX = (script && script.getAttribute('data-site-context')) || '';
  const ACCENT = (script && script.getAttribute('data-accent')) || '#238636';
  const POS = (script && script.getAttribute('data-position')) || 'right';
  const SESSION = 'wgt-' + Math.random().toString(36).slice(2, 9);

  const host = document.createElement('div');
  host.id = 'scraper-bot-widget-host';
  host.style.cssText = `position:fixed;bottom:24px;${POS}:24px;z-index:999999;font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;`;
  document.body.appendChild(host);
  const shadow = host.attachShadow({ mode: 'open' });

  const style = document.createElement('style');
  style.textContent = `
    *{box-sizing:border-box;margin:0;padding:0}
    .fab{min-width:54px;height:42px;padding:0 14px;border-radius:21px;background:${ACCENT};border:none;cursor:pointer;
         display:flex;align-items:center;justify-content:center;font-size:13px;font-weight:600;color:#fff;
         box-shadow:0 4px 16px rgba(0,0,0,0.35);transition:transform .15s,box-shadow .15s;
         margin-left:auto;}
    .fab:hover{transform:scale(1.04);box-shadow:0 6px 20px rgba(0,0,0,0.45)}
    .window{display:none;flex-direction:column;width:340px;height:480px;
            background:#161b22;border:1px solid #30363d;border-radius:12px;
            box-shadow:0 8px 32px rgba(0,0,0,0.5);overflow:hidden;margin-bottom:10px;}
    .window.open{display:flex;}
    .header{display:flex;align-items:center;gap:10px;padding:12px 14px;
            background:#0d1117;border-bottom:1px solid #30363d;}
    .header-icon{width:22px;height:22px;background:${ACCENT};border-radius:4px;
                 display:flex;align-items:center;justify-content:center;font-size:11px;font-weight:700;color:#fff;}
    .header-title{flex:1;font-size:14px;font-weight:600;color:#e6edf3;}
    .close-btn{background:none;border:none;cursor:pointer;color:#8b949e;font-size:18px;line-height:1;}
    .close-btn:hover{color:#e6edf3;}
    .messages{flex:1;overflow-y:auto;padding:14px;display:flex;flex-direction:column;gap:10px;}
    .messages::-webkit-scrollbar{width:4px}
    .messages::-webkit-scrollbar-thumb{background:#30363d;border-radius:4px}
    .bubble{padding:10px 13px;border-radius:10px;font-size:13px;line-height:1.6;max-width:88%;}
    .bubble.user{align-self:flex-end;background:${ACCENT};color:#fff;border-bottom-right-radius:3px;}
    .bubble.bot{align-self:flex-start;background:#21262d;color:#e6edf3;border:1px solid #30363d;border-bottom-left-radius:3px;}
    .bubble.bot a{color:#58a6ff;}
    .typing{display:flex;gap:4px;align-items:center;padding:10px 13px;background:#21262d;
            border:1px solid #30363d;border-radius:10px;align-self:flex-start;border-bottom-left-radius:3px;}
    .dot{width:7px;height:7px;background:#8b949e;border-radius:50%;animation:bonce 1.2s infinite;}
    .dot:nth-child(2){animation-delay:.2s}.dot:nth-child(3){animation-delay:.4s}
    @keyframes bonce{0%,80%,100%{transform:translateY(0)}40%{transform:translateY(-5px)}}
    .input-row{display:flex;gap:8px;padding:10px 12px;background:#0d1117;border-top:1px solid #30363d;}
    .input-row input{flex:1;background:#21262d;border:1px solid #30363d;color:#e6edf3;
                     border-radius:8px;padding:7px 11px;font-size:13px;outline:none;}
    .input-row input:focus{border-color:${ACCENT};}
    .input-row button{background:${ACCENT};border:none;color:#fff;border-radius:8px;
                      padding:7px 13px;cursor:pointer;font-size:13px;font-weight:600;}
    .input-row button:hover{opacity:.88;}
    .powered{text-align:center;font-size:11px;color:#484f58;padding:4px 0 6px;background:#0d1117;}
    .powered a{color:#58a6ff;text-decoration:none;}
  `;
  shadow.appendChild(style);

  const win = document.createElement('div');
  win.className = 'window';
  win.innerHTML = `
    <div class="header">
      <div class="header-icon">AI</div>
      <div class="header-title">${TITLE}</div>
      <button class="close-btn">&times;</button>
    </div>
    <div class="messages" id="wgt-msgs"></div>
    <div class="input-row">
      <input id="wgt-input" type="text" placeholder="Type a message..." autocomplete="off">
      <button id="wgt-send">Send</button>
    </div>
    <div class="powered"><a href="#" target="_blank">ScraperBot</a></div>
  `;

  const fab = document.createElement('button');
  fab.className = 'fab';
  fab.textContent = 'Chat';
  fab.title = 'Open ' + TITLE;

  shadow.appendChild(win);
  shadow.appendChild(fab);

  let open = false;

  fab.addEventListener('click', () => {
    open = !open;
    win.classList.toggle('open', open);
    fab.textContent = open ? 'Close' : 'Chat';
    if (open && shadow.getElementById('wgt-msgs').children.length === 0) {
      addBubble('bot', `Hello. How can I help you?`);
    }
    shadow.getElementById('wgt-input').focus();
  });

  win.querySelector('.close-btn').addEventListener('click', () => {
    open = false;
    win.classList.remove('open');
    fab.textContent = 'Chat';
  });

  function addBubble(role, text) {
    const msgs = shadow.getElementById('wgt-msgs');
    const b = document.createElement('div');
    b.className = 'bubble ' + role;
    b.innerHTML = text.replace(/\n/g, '<br>').replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
    msgs.appendChild(b);
    msgs.scrollTop = msgs.scrollHeight;
    return b;
  }

  function showTyping() {
    const msgs = shadow.getElementById('wgt-msgs');
    const t = document.createElement('div');
    t.className = 'typing';
    t.innerHTML = '<div class="dot"></div><div class="dot"></div><div class="dot"></div>';
    msgs.appendChild(t);
    msgs.scrollTop = msgs.scrollHeight;
    return t;
  }

  function sendMessage() {
    const input = shadow.getElementById('wgt-input');
    const msg = input.value.trim();
    if (!msg) return;
    input.value = '';
    addBubble('user', msg);
    const typing = showTyping();

    fetch(API + '/api/v1/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ message: msg, session_id: SESSION, mode: MODE, site_context: CTX }),
    })
      .then(r => r.json())
      .then(data => {
        typing.remove();
        let reply = data.reply || 'No response received.';
        if (data.sources && data.sources.length) {
          reply += '\n<span style="font-size:11px;color:#8b949e">Sources: ' +
            data.sources.map(s => `<a href="${s.url}" target="_blank">${s.title}</a>`).join(', ') + '</span>';
        }
        addBubble('bot', reply);
      })
      .catch(() => {
        typing.remove();
        addBubble('bot', 'Could not connect to server.');
      });
  }

  shadow.getElementById('wgt-send').addEventListener('click', sendMessage);
  shadow.getElementById('wgt-input').addEventListener('keydown', e => {
    if (e.key === 'Enter') {
      e.preventDefault();
      sendMessage();
    }
  });
})();
