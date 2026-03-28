package setup

const setupHTML = `<!DOCTYPE html>
<html lang="zh">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>EnvNexus Agent Setup</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }
  :root {
    --blue: #3b82f6; --blue-dark: #2563eb; --green: #10b981;
    --bg: #f1f5f9; --card: #ffffff; --border: #e2e8f0;
    --text: #1e293b; --muted: #64748b;
    --font: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  }
  body {
    font-family: var(--font); background: var(--bg); color: var(--text);
    min-height: 100vh; display: flex; align-items: center; justify-content: center;
    padding: 24px;
  }
  .wizard {
    background: var(--card); border-radius: 16px; box-shadow: 0 4px 24px rgba(0,0,0,0.08);
    width: 100%; max-width: 560px; overflow: hidden;
  }
  .wizard-header {
    background: linear-gradient(135deg, #1e40af 0%, #3b82f6 100%);
    color: white; padding: 32px 32px 24px; text-align: center;
  }
  .wizard-header h1 { font-size: 22px; font-weight: 700; margin-bottom: 6px; }
  .wizard-header p { font-size: 13px; opacity: 0.85; }
  .wizard-logo {
    width: 56px; height: 56px; background: rgba(255,255,255,0.15);
    border-radius: 14px; display: flex; align-items: center; justify-content: center;
    margin: 0 auto 16px; font-size: 28px;
  }
  .steps {
    display: flex; justify-content: center; gap: 8px; margin-top: 20px;
  }
  .step-dot {
    width: 10px; height: 10px; border-radius: 50%;
    background: rgba(255,255,255,0.3); transition: all 0.3s;
  }
  .step-dot.active { background: white; transform: scale(1.2); }
  .step-dot.done { background: #86efac; }

  .wizard-body { padding: 28px 32px; }
  .page { display: none; }
  .page.active { display: block; animation: fadeIn 0.3s ease; }
  @keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: none; } }

  .form-group { margin-bottom: 20px; }
  .form-label {
    display: block; font-size: 13px; font-weight: 600; color: var(--text);
    margin-bottom: 6px;
  }
  .form-hint { font-size: 11px; color: var(--muted); margin-top: 4px; }
  .form-input, .form-select {
    width: 100%; padding: 10px 14px; border: 1.5px solid var(--border);
    border-radius: 8px; font-size: 14px; font-family: var(--font);
    background: #f8fafc; outline: none; transition: border-color 0.2s, box-shadow 0.2s;
  }
  .form-input:focus, .form-select:focus {
    border-color: var(--blue); box-shadow: 0 0 0 3px rgba(59,130,246,0.12);
    background: white;
  }
  .form-input.mono { font-family: 'Consolas', 'Monaco', monospace; font-size: 13px; }

  .wizard-footer {
    padding: 0 32px 28px; display: flex; justify-content: space-between; gap: 12px;
  }
  .btn {
    padding: 10px 24px; border-radius: 8px; font-size: 14px; font-weight: 600;
    cursor: pointer; border: none; transition: all 0.15s; font-family: var(--font);
  }
  .btn:hover { transform: translateY(-1px); }
  .btn:active { transform: none; }
  .btn-primary { background: var(--blue); color: white; }
  .btn-primary:hover { background: var(--blue-dark); }
  .btn-secondary { background: var(--border); color: var(--text); }
  .btn-secondary:hover { background: #cbd5e1; }
  .btn-success { background: var(--green); color: white; }
  .btn-success:hover { background: #059669; }
  .btn:disabled { opacity: 0.5; cursor: not-allowed; transform: none; }

  .success-page { text-align: center; padding: 20px 0; }
  .success-icon { font-size: 56px; margin-bottom: 16px; }
  .success-title { font-size: 20px; font-weight: 700; margin-bottom: 8px; color: var(--green); }
  .success-text { font-size: 14px; color: var(--muted); line-height: 1.6; }
  .config-path {
    background: #f1f5f9; border: 1px solid var(--border); border-radius: 6px;
    padding: 8px 12px; font-family: monospace; font-size: 12px;
    margin-top: 12px; word-break: break-all; color: var(--text);
  }

  .test-result {
    margin-top: 12px; padding: 10px 14px; border-radius: 8px; font-size: 13px;
    display: none;
  }
  .test-result.success { display: block; background: #ecfdf5; color: #065f46; border: 1px solid #a7f3d0; }
  .test-result.error { display: block; background: #fef2f2; color: #991b1b; border: 1px solid #fecaca; }
  .test-result.loading { display: block; background: #eff6ff; color: #1e40af; border: 1px solid #bfdbfe; }
</style>
</head>
<body>

<div class="wizard">
  <div class="wizard-header">
    <div class="wizard-logo">&#9881;</div>
    <h1>EnvNexus Agent Setup</h1>
    <p>Configure your agent to connect to the EnvNexus platform</p>
    <div class="steps">
      <div class="step-dot active" id="dot-0"></div>
      <div class="step-dot" id="dot-1"></div>
      <div class="step-dot" id="dot-2"></div>
    </div>
  </div>

  <div class="wizard-body">
    <!-- Page 0: Server Connection -->
    <div class="page active" id="page-0">
      <div class="form-group">
        <label class="form-label">Platform API URL</label>
        <input class="form-input mono" id="platform_url" type="text" placeholder="http://192.168.1.100:8080">
        <div class="form-hint">EnvNexus platform server address. Ask your IT administrator for this URL.</div>
      </div>
      <div class="form-group">
        <label class="form-label">WebSocket Gateway URL</label>
        <input class="form-input mono" id="ws_url" type="text" placeholder="ws://192.168.1.100:8081/ws/v1/sessions">
        <div class="form-hint">Real-time communication gateway. Usually same host, port 8081.</div>
      </div>
      <div id="test-result" class="test-result"></div>
    </div>

    <!-- Page 1: Activation -->
    <div class="page" id="page-1">
      <div class="form-group">
        <label class="form-label">Activation Mode</label>
        <select class="form-select" id="activation_mode">
          <option value="">Skip (no activation required)</option>
          <option value="auto">Auto (use activation key)</option>
          <option value="manual">Manual (admin approves device)</option>
          <option value="both">Both (try auto, fallback to manual)</option>
        </select>
        <div class="form-hint">How this agent authenticates with the platform.</div>
      </div>
      <div class="form-group" id="key-group" style="display:none">
        <label class="form-label">Activation Key</label>
        <input class="form-input mono" id="activation_key" type="text" placeholder="Enter activation key from admin console">
        <div class="form-hint">Provided by your administrator when creating the download package.</div>
      </div>
      <div class="form-group">
        <label class="form-label">Enrollment Token (optional)</label>
        <input class="form-input mono" id="enrollment_token" type="text" placeholder="Optional">
        <div class="form-hint">Only needed for advanced enrollment scenarios.</div>
      </div>
    </div>

    <!-- Page 2: Success -->
    <div class="page" id="page-2">
      <div class="success-page">
        <div class="success-icon">&#10004;</div>
        <div class="success-title">Setup Complete!</div>
        <div class="success-text">
          Your EnvNexus Agent has been configured successfully.<br>
          You can now close this window and start the agent.
        </div>
        <div class="config-path" id="config-path"></div>
      </div>
    </div>
  </div>

  <div class="wizard-footer">
    <button class="btn btn-secondary" id="btn-back" onclick="prevPage()" style="visibility:hidden">Back</button>
    <div style="display:flex; gap:8px;">
      <button class="btn btn-secondary" id="btn-test" onclick="testConnection()" style="display:none">Test Connection</button>
      <button class="btn btn-primary" id="btn-next" onclick="nextPage()">Next</button>
    </div>
  </div>
</div>

<script>
let currentPage = 0;
const totalPages = 3;

async function loadConfig() {
  try {
    const resp = await fetch('/api/config');
    const cfg = await resp.json();
    if (cfg.platform_url) document.getElementById('platform_url').value = cfg.platform_url;
    if (cfg.ws_url) document.getElementById('ws_url').value = cfg.ws_url;
    if (cfg.activation_mode) document.getElementById('activation_mode').value = cfg.activation_mode;
    if (cfg.activation_key) document.getElementById('activation_key').value = cfg.activation_key;
    if (cfg.enrollment_token) document.getElementById('enrollment_token').value = cfg.enrollment_token;
    toggleKeyGroup();
  } catch(e) { console.error('Failed to load config', e); }
}

function toggleKeyGroup() {
  const mode = document.getElementById('activation_mode').value;
  document.getElementById('key-group').style.display =
    (mode === 'auto' || mode === 'both') ? 'block' : 'none';
}
document.getElementById('activation_mode').addEventListener('change', toggleKeyGroup);

function showPage(idx) {
  document.querySelectorAll('.page').forEach((p, i) => {
    p.classList.toggle('active', i === idx);
  });
  document.querySelectorAll('.step-dot').forEach((d, i) => {
    d.className = 'step-dot' + (i === idx ? ' active' : i < idx ? ' done' : '');
  });

  document.getElementById('btn-back').style.visibility = idx > 0 && idx < 2 ? 'visible' : 'hidden';
  document.getElementById('btn-test').style.display = idx === 0 ? 'inline-block' : 'none';

  if (idx === 2) {
    document.getElementById('btn-next').textContent = 'Close';
    document.getElementById('btn-next').className = 'btn btn-success';
  } else {
    document.getElementById('btn-next').textContent = idx === 1 ? 'Save & Finish' : 'Next';
    document.getElementById('btn-next').className = 'btn btn-primary';
  }
}

async function nextPage() {
  if (currentPage === 0) {
    currentPage = 1;
    showPage(1);
  } else if (currentPage === 1) {
    await saveConfig();
    currentPage = 2;
    showPage(2);
  } else if (currentPage === 2) {
    await fetch('/api/done', { method: 'POST' });
    document.querySelector('.wizard-body').innerHTML =
      '<div style="text-align:center; padding:40px; color:#64748b;">You can close this tab now.</div>';
  }
}

function prevPage() {
  if (currentPage > 0 && currentPage < 2) {
    currentPage--;
    showPage(currentPage);
  }
}

async function saveConfig() {
  const body = {
    platform_url: document.getElementById('platform_url').value,
    ws_url: document.getElementById('ws_url').value,
    activation_mode: document.getElementById('activation_mode').value,
    activation_key: document.getElementById('activation_key').value,
    enrollment_token: document.getElementById('enrollment_token').value,
  };
  try {
    const resp = await fetch('/api/save', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    const data = await resp.json();
    if (data.ok) {
      document.getElementById('config-path').textContent = 'Configuration saved';
    }
  } catch(e) {
    alert('Failed to save: ' + e.message);
  }
}

async function testConnection() {
  const el = document.getElementById('test-result');
  const url = document.getElementById('platform_url').value;
  if (!url) { el.className = 'test-result error'; el.textContent = 'Please enter a Platform API URL first.'; return; }

  el.className = 'test-result loading';
  el.textContent = 'Testing connection to ' + url + ' ...';

  try {
    const resp = await fetch(url + '/readyz', { mode: 'no-cors', signal: AbortSignal.timeout(5000) });
    el.className = 'test-result success';
    el.textContent = 'Connection successful! Platform is reachable.';
  } catch(e) {
    el.className = 'test-result error';
    el.textContent = 'Connection failed: ' + (e.message || 'unreachable') + '. Please check the URL and try again.';
  }
}

loadConfig();
</script>
</body>
</html>`
