import {
  app,
  BrowserWindow,
  ipcMain,
  Tray,
  Menu,
  nativeImage,
  NativeImage,
  dialog,
  shell,
} from 'electron';
import * as path from 'path';
import * as http from 'http';
import * as https from 'https';
import * as fs from 'fs';
import * as child_process from 'child_process';
import * as url from 'url';

// ── Types ─────────────────────────────────────────────────────────────────────

interface Settings {
  language: 'zh' | 'en';
  platformURL: string;
  logLevel: 'debug' | 'info' | 'warn' | 'error';
  agentCorePath: string;
  autoStart: boolean;
}

// ── State ──────────────────────────────────────────────────────────────────────

let mainWindow: BrowserWindow | null = null;
let tray: Tray | null = null;
let agentCoreProcess: child_process.ChildProcess | null = null;
let isQuitting = false;
let agentCoreLogs: string[] = [];
const MAX_LOG_LINES = 2000;
let lastSpawnTime = 0;
const MIN_SPAWN_INTERVAL_MS = 3000;

const DEFAULT_SETTINGS: Settings = {
  language: 'zh',
  platformURL: 'http://localhost:8080',
  logLevel: 'info',
  agentCorePath: '',
  autoStart: true,
};

// Portable mode: when a `.portable` marker file exists next to the exe,
// all data (settings, agent data, logs) lives alongside the binary
// instead of in %APPDATA% or system-level directories.
const PORTABLE_BASE_DIR = path.dirname(app.getPath('exe'));
const IS_PORTABLE = fs.existsSync(path.join(PORTABLE_BASE_DIR, '.portable'));

function isPortableMode(): boolean {
  return IS_PORTABLE;
}

function getPortableBaseDir(): string {
  return PORTABLE_BASE_DIR;
}

const SETTINGS_FILE = isPortableMode()
  ? path.join(getPortableBaseDir(), 'settings.json')
  : path.join(app.getPath('userData'), 'settings.json');

function getAgentDataDir(): string {
  if (isPortableMode()) {
    const dataDir = path.join(getPortableBaseDir(), 'data');
    fs.mkdirSync(dataDir, { recursive: true });
    return dataDir;
  }

  const appDir = path.dirname(app.getPath('exe'));
  const dataDir = path.join(appDir, 'data');
  try {
    fs.mkdirSync(dataDir, { recursive: true });
    fs.accessSync(dataDir, fs.constants.W_OK);
    return dataDir;
  } catch {
    const homeDir = process.env.HOME || process.env.USERPROFILE || '';
    return path.join(homeDir, '.envnexus', 'agent');
  }
}

// ── Settings helpers ───────────────────────────────────────────────────────────

function loadSettings(): Settings {
  try {
    if (fs.existsSync(SETTINGS_FILE)) {
      const raw = fs.readFileSync(SETTINGS_FILE, 'utf-8');
      return { ...DEFAULT_SETTINGS, ...JSON.parse(raw) };
    }
  } catch {
    // ignore
  }
  return { ...DEFAULT_SETTINGS };
}

function saveSettings(settings: Settings): void {
  try {
    fs.mkdirSync(path.dirname(SETTINGS_FILE), { recursive: true });
    fs.writeFileSync(SETTINGS_FILE, JSON.stringify(settings, null, 2), 'utf-8');
  } catch (e) {
    console.error('Failed to save settings', e);
  }
}

// ── Local API helper ───────────────────────────────────────────────────────────

function localAPIRequest(method: string, path: string, body?: object): Promise<any> {
  return new Promise((resolve, reject) => {
    const postData = body ? JSON.stringify(body) : undefined;
    const options: http.RequestOptions = {
      hostname: '127.0.0.1',
      port: 17700,
      path,
      method,
      headers: {
        'Content-Type': 'application/json',
        ...(postData ? { 'Content-Length': Buffer.byteLength(postData) } : {}),
      },
    };

    const req = http.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        try {
          resolve(JSON.parse(data));
        } catch {
          resolve({ raw: data });
        }
      });
    });

    req.on('error', (e) => reject(new Error(`agent-core not reachable: ${e.message}`)));
    req.setTimeout(5000, () => {
      req.destroy();
      reject(new Error('agent-core did not respond in 5s'));
    });

    if (postData) req.write(postData);
    req.end();
  });
}

// Platform API helper (respects settings)
function platformAPIRequest(endpoint: string, token: string): Promise<any> {
  const settings = loadSettings();
  return new Promise((resolve, reject) => {
    const parsed = new url.URL(endpoint, settings.platformURL);
    const isHTTPS = parsed.protocol === 'https:';
    const lib = isHTTPS ? https : http;

    const options = {
      hostname: parsed.hostname,
      port: parsed.port || (isHTTPS ? 443 : 80),
      path: parsed.pathname + parsed.search,
      method: 'GET',
      headers: { 'Authorization': `Bearer ${token}`, 'Content-Type': 'application/json' },
    };

    const req = lib.request(options, (res) => {
      let data = '';
      res.on('data', (chunk) => { data += chunk; });
      res.on('end', () => {
        try { resolve(JSON.parse(data)); }
        catch { resolve({ raw: data }); }
      });
    });
    req.on('error', (e) => reject(e));
    req.setTimeout(10000, () => { req.destroy(); reject(new Error('timeout')); });
    req.end();
  });
}

// ── Agent Core process management ─────────────────────────────────────────────

function spawnAgentCore(): void {
  if (agentCoreProcess) return;

  const now = Date.now();
  if (now - lastSpawnTime < MIN_SPAWN_INTERVAL_MS) {
    console.warn('spawnAgentCore called too quickly, throttling');
    return;
  }
  lastSpawnTime = now;

  const settings = loadSettings();
  const binaryPath = settings.agentCorePath || findAgentCoreBinary();

  if (!binaryPath || !fs.existsSync(binaryPath)) {
    console.warn('agent-core binary not found, skipping spawn');
    appendAgentLog('[WARN] enx-agent binary not found. It will be auto-detected from common paths. If the problem persists, set the path manually in Settings.');
    return;
  }

  const resolvedBinary = path.resolve(binaryPath).toLowerCase();
  const selfExe = path.resolve(app.getPath('exe')).toLowerCase();
  if (resolvedBinary === selfExe) {
    console.error('SAFETY: agentCorePath points to the Electron app itself — aborting spawn to prevent infinite loop');
    appendAgentLog('[ERROR] agentCorePath points to the Electron app itself. Please set the correct enx-agent binary path in Settings.');
    return;
  }

  const baseName = path.basename(resolvedBinary).toLowerCase();
  if (baseName.includes('envnexus') || baseName.includes('electron')) {
    console.error('SAFETY: agentCorePath appears to be an Electron/desktop binary, not enx-agent — aborting spawn');
    appendAgentLog(`[ERROR] Binary "${baseName}" does not look like enx-agent. Please check Settings > Agent Core path.`);
    return;
  }

  console.log('Spawning agent-core:', binaryPath);

  const agentDataDir = getAgentDataDir();
  const agentEnv: Record<string, string> = {
    ...process.env as Record<string, string>,
    ENX_LOG_LEVEL: settings.logLevel,
  };
  if (settings.platformURL) {
    agentEnv.ENX_PLATFORM_URL = settings.platformURL;
  }

  const enxCfg = readAgentEnxConfig(agentDataDir);
  if (enxCfg.enrollment_token) {
    agentEnv.ENX_ENROLLMENT_TOKEN = enxCfg.enrollment_token;
  }
  if (enxCfg.ws_url && !agentEnv.ENX_WS_URL) {
    agentEnv.ENX_WS_URL = enxCfg.ws_url;
  }

  const args: string[] = ['--data-dir', agentDataDir];
  if (settings.platformURL) {
    args.push('--platform-url', settings.platformURL);
  }

  console.log('Agent data dir:', agentDataDir);
  agentCoreProcess = child_process.spawn(binaryPath, args, {
    detached: false,
    stdio: 'pipe',
    env: agentEnv,
  });

  agentCoreProcess.stdout?.on('data', (data) => {
    const line = data.toString().trim();
    console.log('[agent-core]', line);
    appendAgentLog(line);
  });

  agentCoreProcess.stderr?.on('data', (data) => {
    const line = data.toString().trim();
    console.error('[agent-core]', line);
    appendAgentLog(line);
  });

  agentCoreProcess.on('exit', (code) => {
    console.log('agent-core exited with code', code);
    agentCoreProcess = null;
    if (!isQuitting) {
      updateTrayStatus('offline');
    }
  });

  agentCoreProcess.on('error', (err) => {
    console.error('Failed to start agent-core:', err);
    agentCoreProcess = null;
  });
}

function findAgentCoreBinary(): string {
  const isWin = process.platform === 'win32';
  const binaryName = isWin ? 'enx-agent.exe' : 'enx-agent';
  const appExeDir = path.dirname(app.getPath('exe'));
  const homeDir = process.env.HOME || process.env.USERPROFILE || '';

  const candidates = [
    // 1. Electron packaged app: extraResources/bin/
    path.join(process.resourcesPath || '', 'bin', binaryName),
    // 2. Same directory as the desktop app exe
    path.join(appExeDir, binaryName),
    // 3. bin/ subfolder next to the desktop app
    path.join(appExeDir, 'bin', binaryName),
    // 4. Agent data directory (where agent.enx lives)
    path.join(getAgentDataDir(), binaryName),
    // 5. Development: project root bin/
    path.join(app.getAppPath(), '..', '..', 'bin', binaryName),
    // 6. Current working directory
    path.join(process.cwd(), binaryName),
    path.join(process.cwd(), 'bin', binaryName),
    // 7. User's Downloads folder (common after extracting ZIP)
    path.join(app.getPath('downloads'), binaryName),
  ];

  if (isWin) {
    // 8. Common Windows install locations
    candidates.push(path.join(homeDir, '.envnexus', 'agent', binaryName));
    candidates.push(path.join(homeDir, '.envnexus', binaryName));
    // 9. Search extracted ZIP folders in Downloads (EnvNexus-Agent-windows-*)
    try {
      const dlDir = app.getPath('downloads');
      const entries = fs.readdirSync(dlDir);
      for (const entry of entries) {
        if (entry.startsWith('EnvNexus-Agent-') && fs.statSync(path.join(dlDir, entry)).isDirectory()) {
          candidates.push(path.join(dlDir, entry, binaryName));
        }
      }
    } catch {}
  } else {
    candidates.push('/usr/local/bin/enx-agent');
    candidates.push(path.join(homeDir, '.local', 'bin', 'enx-agent'));
    candidates.push(path.join(homeDir, '.envnexus', 'agent', binaryName));
    candidates.push(path.join(homeDir, '.envnexus', binaryName));
  }

  const found = candidates.find((p) => {
    try { return fs.existsSync(p); } catch { return false; }
  });

  if (found) {
    console.log('[auto-detect] Found enx-agent at:', found);
  } else {
    console.warn('[auto-detect] enx-agent not found in any candidate path');
    candidates.forEach(c => console.log('  checked:', c));
  }

  return found || '';
}

function stopAgentCore(): void {
  if (!agentCoreProcess) return;
  const proc = agentCoreProcess;
  agentCoreProcess = null;
  try {
    if (process.platform === 'win32') {
      child_process.execSync(`taskkill /pid ${proc.pid} /f /t`, { stdio: 'ignore' });
    } else {
      proc.kill('SIGTERM');
    }
  } catch {
    try { proc.kill('SIGKILL'); } catch {}
  }
}

function appendAgentLog(line: string): void {
  const timestamp = new Date().toISOString().substring(11, 19);
  for (const l of line.split('\n')) {
    if (l.trim()) {
      agentCoreLogs.push(`[${timestamp}] ${l}`);
    }
  }
  while (agentCoreLogs.length > MAX_LOG_LINES) {
    agentCoreLogs.shift();
  }
  mainWindow?.webContents.send('agent-core-log', line);
}

// ── Tray ───────────────────────────────────────────────────────────────────────

type ConnectionStatus = 'online' | 'offline' | 'connecting';

let currentStatus: ConnectionStatus = 'connecting';

function createTrayIcon(status: ConnectionStatus): NativeImage {
  const statusColors: Record<ConnectionStatus, string> = {
    online: '#10b981',
    offline: '#9ca3af',
    connecting: '#f59e0b',
  };
  const dotColor = statusColors[status];

  const size = 32;
  const buf = Buffer.alloc(size * size * 4, 0);
  const brandR = 99, brandG = 102, brandB = 241;
  const [dotR, dotG, dotB] = hexToRGB(dotColor);
  const cr = 6;

  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const idx = (y * size + x) * 4;

      if (isInsideRoundedRect(x, y, 2, 2, 26, 26, cr)) {
        buf[idx] = brandR; buf[idx + 1] = brandG; buf[idx + 2] = brandB; buf[idx + 3] = 255;
        const lx = x - 8, ly = y - 7;
        if (lx >= 0 && lx < 14 && ly >= 0 && ly < 18) {
          const isE = (lx >= 2 && lx <= 4 && ly >= 1 && ly <= 16) ||
                      (lx >= 2 && lx <= 11 && ly >= 1 && ly <= 3) ||
                      (lx >= 2 && lx <= 10 && ly >= 7 && ly <= 9) ||
                      (lx >= 2 && lx <= 11 && ly >= 14 && ly <= 16);
          if (isE) {
            buf[idx] = 255; buf[idx + 1] = 255; buf[idx + 2] = 255; buf[idx + 3] = 255;
          }
        }
      }

      const ddx = x - 25, ddy = y - 25;
      const dotDist = Math.sqrt(ddx * ddx + ddy * ddy);
      if (dotDist <= 4.5) {
        buf[idx] = dotR; buf[idx + 1] = dotG; buf[idx + 2] = dotB; buf[idx + 3] = 255;
      } else if (dotDist <= 6) {
        buf[idx] = 255; buf[idx + 1] = 255; buf[idx + 2] = 255; buf[idx + 3] = 255;
      }
    }
  }

  return nativeImage.createFromBuffer(buf, { width: size, height: size });
}

function hexToRGB(hex: string): [number, number, number] {
  const h = hex.replace('#', '');
  return [parseInt(h.substring(0, 2), 16), parseInt(h.substring(2, 4), 16), parseInt(h.substring(4, 6), 16)];
}

function trayText(key: string): string {
  const lang = loadSettings().language || 'zh';
  const dict: Record<string, Record<string, string>> = {
    zh: {
      tooltip_online: 'EnvNexus Agent — 在线',
      tooltip_offline: 'EnvNexus Agent — 离线',
      tooltip_connecting: 'EnvNexus Agent — 连接中...',
      status_online: '● 在线',
      status_offline: '○ 离线',
      status_connecting: '◌ 连接中...',
      open_panel: '打开控制面板',
      restart_core: '重启 Agent Core',
      start_core: '启动 Agent Core',
      quit: '退出',
    },
    en: {
      tooltip_online: 'EnvNexus Agent — Online',
      tooltip_offline: 'EnvNexus Agent — Offline',
      tooltip_connecting: 'EnvNexus Agent — Connecting...',
      status_online: '● Online',
      status_offline: '○ Offline',
      status_connecting: '◌ Connecting...',
      open_panel: 'Open Dashboard',
      restart_core: 'Restart Agent Core',
      start_core: 'Start Agent Core',
      quit: 'Quit',
    },
  };
  return (dict[lang] && dict[lang][key]) || dict.zh[key] || key;
}

function updateTrayStatus(status: ConnectionStatus): void {
  currentStatus = status;
  if (!tray) return;
  tray.setImage(createTrayIcon(status));
  tray.setToolTip(trayText(`tooltip_${status}`));
  buildTrayMenu();
}

function buildTrayMenu(): void {
  if (!tray) return;

  const statusLabel = trayText(`status_${currentStatus}`);

  const menu = Menu.buildFromTemplate([
    { label: `EnvNexus Agent  ${statusLabel}`, enabled: false },
    { type: 'separator' },
    {
      label: trayText('open_panel'),
      click: () => {
        mainWindow?.show();
        mainWindow?.focus();
      },
    },
    {
      label: agentCoreProcess ? trayText('restart_core') : trayText('start_core'),
      click: () => {
        stopAgentCore();
        lastSpawnTime = 0;
        setTimeout(spawnAgentCore, 1500);
      },
    },
    { type: 'separator' },
    {
      label: trayText('quit'),
      click: () => {
        isQuitting = true;
        stopAgentCore();
        app.quit();
      },
    },
  ]);

  tray.setContextMenu(menu);
}

function createTray(): void {
  tray = new Tray(createTrayIcon('connecting'));
  tray.setToolTip('EnvNexus Agent');
  buildTrayMenu();

  tray.on('click', () => {
    if (mainWindow) {
      if (mainWindow.isVisible()) {
        mainWindow.hide();
      } else {
        mainWindow.show();
        mainWindow.focus();
      }
    }
  });
}

// ── Window ─────────────────────────────────────────────────────────────────────

function createAppIcon(): NativeImage {
  const size = 256;
  const buf = Buffer.alloc(size * size * 4, 0);
  const brandR = 99, brandG = 102, brandB = 241;
  const cornerR = size * 0.22;

  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const idx = (y * size + x) * 4;
      if (isInsideRoundedRect(x, y, 4, 4, size - 8, size - 8, cornerR)) {
        buf[idx] = brandR; buf[idx + 1] = brandG; buf[idx + 2] = brandB; buf[idx + 3] = 255;

        const lx = x - size * 0.28;
        const ly = y - size * 0.22;
        const ew = size * 0.44;
        const eh = size * 0.56;
        const barW = ew * 0.18;
        const hBarH = eh * 0.14;

        const isE = (lx >= 0 && lx <= barW && ly >= 0 && ly <= eh) ||
                    (lx >= 0 && lx <= ew && ly >= 0 && ly <= hBarH) ||
                    (lx >= 0 && lx <= ew * 0.85 && ly >= eh * 0.43 && ly <= eh * 0.43 + hBarH) ||
                    (lx >= 0 && lx <= ew && ly >= eh - hBarH && ly <= eh);

        if (isE) {
          buf[idx] = 255; buf[idx + 1] = 255; buf[idx + 2] = 255; buf[idx + 3] = 255;
        }
      }
    }
  }
  return nativeImage.createFromBuffer(buf, { width: size, height: size });
}

function isInsideRoundedRect(px: number, py: number, rx: number, ry: number, rw: number, rh: number, cr: number): boolean {
  const x = px - rx;
  const y = py - ry;
  if (x < 0 || x > rw || y < 0 || y > rh) return false;
  if (x < cr && y < cr) {
    const dx = x - cr, dy = y - cr;
    return dx * dx + dy * dy <= cr * cr;
  }
  if (x > rw - cr && y < cr) {
    const dx = x - (rw - cr), dy = y - cr;
    return dx * dx + dy * dy <= cr * cr;
  }
  if (x < cr && y > rh - cr) {
    const dx = x - cr, dy = y - (rh - cr);
    return dx * dx + dy * dy <= cr * cr;
  }
  if (x > rw - cr && y > rh - cr) {
    const dx = x - (rw - cr), dy = y - (rh - cr);
    return dx * dx + dy * dy <= cr * cr;
  }
  return true;
}

function createWindow(): void {
  const appIcon = createAppIcon();

  mainWindow = new BrowserWindow({
    width: 960,
    height: 680,
    minWidth: 720,
    minHeight: 500,
    title: 'EnvNexus Agent',
    icon: appIcon,
    webPreferences: {
      preload: path.join(__dirname, '../preload/preload.js'),
      nodeIntegration: false,
      contextIsolation: true,
    },
    show: false,
    backgroundColor: '#f8fafc',
  });

  mainWindow.loadFile(path.join(__dirname, '../../src/renderer/index.html'));

  mainWindow.once('ready-to-show', () => {
    mainWindow?.show();
  });

  mainWindow.on('close', (event) => {
    if (!isQuitting) {
      event.preventDefault();
      mainWindow?.hide();
    }
  });

  mainWindow.on('closed', () => {
    mainWindow = null;
  });
}

// ── IPC Handlers ──────────────────────────────────────────────────────────────

function safeLocalAPI(method: string, path: string, body?: object): Promise<any> {
  return localAPIRequest(method, path, body).catch((err: Error) => ({
    error: err.message || 'agent-core not reachable',
  }));
}

function registerIPC(): void {
  ipcMain.handle('get-agent-status', () =>
    safeLocalAPI('GET', '/local/v1/runtime/status')
  );

  ipcMain.handle('get-pending-approvals', () =>
    safeLocalAPI('GET', '/local/v1/approvals/pending')
  );

  ipcMain.handle('resolve-approval', (_e, id: string, approved: boolean) =>
    safeLocalAPI('POST', `/local/v1/approvals/${id}/resolve`, { approved })
  );

  ipcMain.handle('export-diagnostics', () =>
    safeLocalAPI('POST', '/local/v1/diagnostics/export', {})
  );

  ipcMain.handle('send-diagnose', (_e, query: string, history: any[]) =>
    safeLocalAPI('POST', '/local/v1/diagnose', { intent: query, history })
  );

  ipcMain.handle('get-settings', () => loadSettings());

  ipcMain.handle('save-settings', (_e, settings: Settings) => {
    saveSettings(settings);
    return { ok: true };
  });

  ipcMain.handle('get-connection-status', () => currentStatus);

  ipcMain.handle('open-external', (_e, url: string) => {
    shell.openExternal(url);
  });

  ipcMain.handle('choose-agent-binary', async () => {
    if (!mainWindow) return null;
    const settings = loadSettings();
    const title = settings.language === 'en' ? 'Select enx-agent executable' : '选择 enx-agent 可执行文件';
    const result = await dialog.showOpenDialog(mainWindow, {
      title,
      properties: ['openFile'],
      filters: [{ name: 'Executable', extensions: ['', 'exe'] }],
    });
    return result.canceled ? null : result.filePaths[0];
  });

  ipcMain.handle('restart-agent-core', () => {
    stopAgentCore();
    lastSpawnTime = 0;
    setTimeout(spawnAgentCore, 1500);
    return { ok: true };
  });

  ipcMain.handle('get-app-version', () => app.getVersion());

  ipcMain.handle('get-recent-sessions', () =>
    safeLocalAPI('GET', '/local/v1/sessions/recent')
  );

  ipcMain.handle('get-agent-core-logs', () => agentCoreLogs.join('\n'));

  ipcMain.handle('get-detected-agent-path', () => {
    const settings = loadSettings();
    if (settings.agentCorePath) return settings.agentCorePath;
    return findAgentCoreBinary();
  });
}

// ── Health polling ─────────────────────────────────────────────────────────────

function startHealthPolling(): void {
  setInterval(async () => {
    try {
      await localAPIRequest('GET', '/local/v1/runtime/status');
      if (currentStatus !== 'online') updateTrayStatus('online');
    } catch {
      if (currentStatus !== 'offline') updateTrayStatus('offline');
    }
  }, 10_000);
}

// ── Auto-update ───────────────────────────────────────────────────────────────

function initAutoUpdate(): void {
  try {
    const { autoUpdater } = require('electron-updater');

    autoUpdater.autoDownload = false;
    autoUpdater.autoInstallOnAppQuit = true;

    autoUpdater.on('update-available', (info: any) => {
      console.log(`[updater] Update available: ${info.version}`);
      if (mainWindow) {
        mainWindow.webContents.send('update-available', info.version);
      }
      autoUpdater.downloadUpdate();
    });

    autoUpdater.on('update-downloaded', (info: any) => {
      console.log(`[updater] Update downloaded: ${info.version}`);
      if (mainWindow) {
        mainWindow.webContents.send('update-downloaded', info.version);
      }
    });

    autoUpdater.on('error', (err: Error) => {
      console.error('[updater] Error:', err.message);
    });

    autoUpdater.checkForUpdates().catch(() => {});
    setInterval(() => {
      autoUpdater.checkForUpdates().catch(() => {});
    }, 4 * 60 * 60 * 1000);
  } catch {
    console.log('[updater] electron-updater not available (dev mode)');
  }
}

// ── Read agent.enx config from data dir and bundled locations ─────────────────

function readAgentEnxConfig(agentDataDir: string): Record<string, string> {
  const exeDir = path.dirname(app.getPath('exe'));
  const searchPaths = [
    path.join(agentDataDir, 'agent.enx'),
    path.join(exeDir, 'agent.enx'),
    path.join(process.resourcesPath || '', 'agent.enx'),
    path.join(exeDir, '..', 'agent.enx'),
  ];
  const merged: Record<string, string> = {};
  for (const p of searchPaths) {
    try {
      if (!fs.existsSync(p)) continue;
      const cfg = parseTOMLConfig(fs.readFileSync(p, 'utf-8'));
      for (const [k, v] of Object.entries(cfg)) {
        if (v && !merged[k]) merged[k] = v;
      }
    } catch {}
  }
  return merged;
}

// ── Config sync from agent.enx ────────────────────────────────────────────────

function syncSettingsFromEnxConfig(): void {
  const agentDataDir = getAgentDataDir();
  const exeDir = path.dirname(app.getPath('exe'));
  const enxPaths = [
    path.join(agentDataDir, 'agent.enx'),
    path.join(exeDir, 'agent.enx'),
    path.join(process.resourcesPath || '', 'agent.enx'),
    path.join(exeDir, '..', 'agent.enx'),
  ];

  for (const enxPath of enxPaths) {
    try {
      if (!fs.existsSync(enxPath)) continue;
      const cfg = parseTOMLConfig(fs.readFileSync(enxPath, 'utf-8'));
      if (!cfg.platform_url) continue;

      const settings = loadSettings();
      const isDefault = !settings.platformURL || settings.platformURL === DEFAULT_SETTINGS.platformURL;
      if (isDefault && cfg.platform_url !== settings.platformURL) {
        settings.platformURL = cfg.platform_url;
        saveSettings(settings);
        console.log('[config-sync] Updated platformURL from agent.enx:', cfg.platform_url);
      }
      return;
    } catch {}
  }
}

// ── First-launch config import ────────────────────────────────────────────────

function parseTOMLConfig(content: string): Record<string, string> {
  const result: Record<string, string> = {};
  for (const line of content.split('\n')) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const eqIdx = trimmed.indexOf('=');
    if (eqIdx === -1) continue;
    const key = trimmed.substring(0, eqIdx).trim();
    let val = trimmed.substring(eqIdx + 1).trim();
    if ((val.startsWith('"') && val.endsWith('"')) || (val.startsWith("'") && val.endsWith("'"))) {
      val = val.slice(1, -1);
    }
    result[key] = val;
  }
  return result;
}

function importBundledConfig(): void {
  const agentDataDir = getAgentDataDir();
  const enxTarget = path.join(agentDataDir, 'agent.enx');
  const jsonTarget = path.join(agentDataDir, 'agent_config.json');
  const targetExists = fs.existsSync(enxTarget) || fs.existsSync(jsonTarget);

  const exeDir = path.dirname(app.getPath('exe'));
  const enxSearchPaths = [
    path.join(exeDir, 'agent.enx'),
    path.join(process.resourcesPath || '', 'agent.enx'),
    path.join(app.getAppPath(), 'agent.enx'),
    path.join(app.getPath('downloads'), 'agent.enx'),
    path.join(exeDir, '..', 'agent.enx'),
  ];
  console.log('[config] Searching for agent.enx in:', enxSearchPaths.filter(p => { try { return fs.existsSync(p); } catch { return false; } }));

  for (const src of enxSearchPaths) {
    try {
      if (!fs.existsSync(src)) continue;
      const srcCfg = parseTOMLConfig(fs.readFileSync(src, 'utf-8'));

      if (!targetExists) {
        fs.mkdirSync(agentDataDir, { recursive: true });
        fs.copyFileSync(src, enxTarget);
        console.log('[config] Imported bundled agent.enx from:', src);
      } else if (fs.existsSync(enxTarget)) {
        const existingCfg = parseTOMLConfig(fs.readFileSync(enxTarget, 'utf-8'));
        let merged = false;
        const mergeKeys = ['enrollment_token', 'ws_url', 'activation_mode', 'activation_key'];
        for (const key of mergeKeys) {
          if (srcCfg[key] && !existingCfg[key]) {
            existingCfg[key] = srcCfg[key];
            merged = true;
          }
        }
        if (merged) {
          const lines = ['# EnvNexus Agent Configuration'];
          for (const [k, v] of Object.entries(existingCfg)) {
            lines.push(`${k} = "${v}"`);
          }
          fs.writeFileSync(enxTarget, lines.join('\n') + '\n', 'utf-8');
          console.log('[config] Merged missing fields from bundled agent.enx into existing config');
        }
      }

      if (srcCfg.platform_url) {
        const settings = loadSettings();
        const isDefault = !settings.platformURL || settings.platformURL === DEFAULT_SETTINGS.platformURL;
        if (isDefault) {
          settings.platformURL = srcCfg.platform_url;
          saveSettings(settings);
          console.log('[config] Updated desktop settings with platform URL:', srcCfg.platform_url);
        }
      }
      return;
    } catch (e) {
      console.warn('[config] Failed to import .enx from', src, e);
    }
  }

  if (targetExists) return;

  const jsonSearchPaths = [
    path.join(path.dirname(app.getPath('exe')), 'config.json'),
    path.join(process.resourcesPath || '', 'config.json'),
    path.join(app.getAppPath(), 'config.json'),
    path.join(app.getPath('downloads'), 'config.json'),
  ];

  for (const src of jsonSearchPaths) {
    try {
      if (fs.existsSync(src)) {
        fs.mkdirSync(agentDataDir, { recursive: true });
        fs.copyFileSync(src, jsonTarget);
        console.log('[config] Imported bundled config.json from:', src);

        const cfg = JSON.parse(fs.readFileSync(src, 'utf-8'));
        if (cfg.platform_url) {
          const settings = loadSettings();
          settings.platformURL = cfg.platform_url;
          saveSettings(settings);
          console.log('[config] Updated desktop settings with platform URL:', cfg.platform_url);
        }
        return;
      }
    } catch (e) {
      console.warn('[config] Failed to import from', src, e);
    }
  }
}

// ── Single instance lock ────────────────────────────────────────────────────────

const gotTheLock = app.requestSingleInstanceLock();

if (!gotTheLock) {
  app.quit();
} else {
  app.on('second-instance', () => {
    if (mainWindow) {
      if (mainWindow.isMinimized()) mainWindow.restore();
      mainWindow.show();
      mainWindow.focus();
    }
  });
}

// ── App lifecycle ──────────────────────────────────────────────────────────────

app.whenReady().then(() => {
  importBundledConfig();
  syncSettingsFromEnxConfig();
  registerIPC();
  createTray();
  createWindow();
  startHealthPolling();
  initAutoUpdate();

  const settings = loadSettings();
  if (settings.autoStart) {
    spawnAgentCore();
  }

  app.on('activate', () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      createWindow();
    } else {
      mainWindow?.show();
    }
  });
});

app.on('window-all-closed', () => {
  // Keep running in tray on macOS/Windows
});

app.on('before-quit', () => {
  isQuitting = true;
  stopAgentCore();
});
