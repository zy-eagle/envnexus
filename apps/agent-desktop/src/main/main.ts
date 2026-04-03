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
  maxIterations: number;
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
  maxIterations: 10,
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

function localAPIRequest(method: string, path: string, body?: object, timeoutMs = 5000): Promise<any> {
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
    req.setTimeout(timeoutMs, () => {
      req.destroy();
      reject(new Error(`agent-core did not respond in ${timeoutMs / 1000}s`));
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

const MIN_AGENT_BINARY_SIZE = 12_000;

/** Reject empty/corrupt files that existSync would still accept (common right after a failed or partial update). */
function isLikelyValidAgentBinary(absPath: string): boolean {
  try {
    const st = fs.statSync(absPath);
    if (!st.isFile() || st.size < MIN_AGENT_BINARY_SIZE) return false;
    if (process.platform !== 'win32') return true;
    const fd = fs.openSync(absPath, 'r');
    try {
      const buf = Buffer.alloc(2);
      fs.readSync(fd, buf, 0, 2, 0);
      return buf[0] === 0x4d && buf[1] === 0x5a;
    } finally {
      fs.closeSync(fd);
    }
  } catch {
    return false;
  }
}

/**
 * Windows often reports libuv errno as Error: spawn UNKNOWN for invalid PEs, torn files, or spawn quirks.
 * execFile + cwd next to the binary is more reliable; always catch synchronous throws so the UI does not crash.
 */
function startAgentCoreChild(
  exePath: string,
  args: string[],
  env: Record<string, string>,
): child_process.ChildProcess | null {
  const cwd = path.dirname(exePath);
  const opts: child_process.SpawnOptions = {
    detached: false,
    stdio: 'pipe',
    env,
    windowsHide: process.platform === 'win32',
    cwd: fs.existsSync(cwd) ? cwd : undefined,
  };

  try {
    return child_process.spawn(exePath, args, opts);
  } catch (e) {
    console.warn('[agent-core] spawn() failed, trying execFile():', e);
    try {
      return child_process.execFile(exePath, args, opts);
    } catch (e2) {
      console.error('[agent-core] execFile() failed:', e2);
      appendAgentLog(`[ERROR] Could not start enx-agent: ${(e2 as Error).message}. Check that the file is a valid Windows executable (not blocked or partially updated).`);
      return null;
    }
  }
}

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

  if (!binaryPath) {
    console.warn('agent-core binary not found, skipping spawn');
    appendAgentLog('[WARN] enx-agent binary not found. It will be auto-detected from common paths. If the problem persists, set the path manually in Settings.');
    return;
  }

  const resolvedBinary = path.resolve(binaryPath);
  if (!isLikelyValidAgentBinary(resolvedBinary)) {
    console.warn('agent-core path is missing or not a valid binary:', resolvedBinary);
    appendAgentLog('[WARN] enx-agent is missing or does not look like a valid executable (e.g. still updating or wrong path). Verify resources/bin/enx-agent.exe or set the path in Settings.');
    return;
  }

  const resolvedBinaryLower = resolvedBinary.toLowerCase();
  const selfExe = path.resolve(app.getPath('exe')).toLowerCase();
  if (resolvedBinaryLower === selfExe) {
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

  console.log('Spawning agent-core:', resolvedBinary);

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
  const child = startAgentCoreChild(resolvedBinary, args, agentEnv);
  if (!child) {
    return;
  }
  agentCoreProcess = child;

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
    try {
      const abs = path.resolve(p);
      return isLikelyValidAgentBinary(abs);
    } catch {
      return false;
    }
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

function menuText(key: string): string {
  const lang = loadSettings().language || 'zh';
  const dict: Record<string, Record<string, string>> = {
    zh: {
      file: '文件',
      quit: '退出',
      edit: '编辑',
      undo: '撤销',
      redo: '重做',
      cut: '剪切',
      copy: '复制',
      paste: '粘贴',
      select_all: '全选',
      view: '视图',
      reload: '重新加载',
      force_reload: '强制重新加载',
      toggle_devtools: '开发者工具',
      reset_zoom: '重置缩放',
      zoom_in: '放大',
      zoom_out: '缩小',
      fullscreen: '全屏',
      window: '窗口',
      minimize: '最小化',
      close: '关闭',
      help: '帮助',
      about: '关于 EnvNexus Agent',
      version: '版本',
      homepage: '项目主页',
    },
    en: {
      file: 'File',
      quit: 'Quit',
      edit: 'Edit',
      undo: 'Undo',
      redo: 'Redo',
      cut: 'Cut',
      copy: 'Copy',
      paste: 'Paste',
      select_all: 'Select All',
      view: 'View',
      reload: 'Reload',
      force_reload: 'Force Reload',
      toggle_devtools: 'Developer Tools',
      reset_zoom: 'Reset Zoom',
      zoom_in: 'Zoom In',
      zoom_out: 'Zoom Out',
      fullscreen: 'Fullscreen',
      window: 'Window',
      minimize: 'Minimize',
      close: 'Close',
      help: 'Help',
      about: 'About EnvNexus Agent',
      version: 'Version',
      homepage: 'Project Homepage',
    },
  };
  return (dict[lang] && dict[lang][key]) || dict.en[key] || key;
}

function buildAppMenu(): void {
  const isMac = process.platform === 'darwin';

  const template: Electron.MenuItemConstructorOptions[] = [
    {
      label: menuText('file'),
      submenu: [
        {
          label: menuText('quit'),
          accelerator: isMac ? 'Cmd+Q' : 'Alt+F4',
          click: () => { isQuitting = true; stopAgentCore(); app.quit(); },
        },
      ],
    },
    {
      label: menuText('edit'),
      submenu: [
        { label: menuText('undo'), role: 'undo', accelerator: 'CmdOrCtrl+Z' },
        { label: menuText('redo'), role: 'redo', accelerator: 'Shift+CmdOrCtrl+Z' },
        { type: 'separator' },
        { label: menuText('cut'), role: 'cut', accelerator: 'CmdOrCtrl+X' },
        { label: menuText('copy'), role: 'copy', accelerator: 'CmdOrCtrl+C' },
        { label: menuText('paste'), role: 'paste', accelerator: 'CmdOrCtrl+V' },
        { type: 'separator' },
        { label: menuText('select_all'), role: 'selectAll', accelerator: 'CmdOrCtrl+A' },
      ],
    },
    {
      label: menuText('view'),
      submenu: [
        { label: menuText('reload'), role: 'reload', accelerator: 'CmdOrCtrl+R' },
        { label: menuText('force_reload'), role: 'forceReload', accelerator: 'Shift+CmdOrCtrl+R' },
        { label: menuText('toggle_devtools'), role: 'toggleDevTools', accelerator: 'F12' },
        { type: 'separator' },
        { label: menuText('reset_zoom'), role: 'resetZoom', accelerator: 'CmdOrCtrl+0' },
        { label: menuText('zoom_in'), role: 'zoomIn', accelerator: 'CmdOrCtrl+=' },
        { label: menuText('zoom_out'), role: 'zoomOut', accelerator: 'CmdOrCtrl+-' },
        { type: 'separator' },
        { label: menuText('fullscreen'), role: 'togglefullscreen', accelerator: 'F11' },
      ],
    },
    {
      label: menuText('window'),
      submenu: [
        { label: menuText('minimize'), role: 'minimize' },
        { label: menuText('close'), role: 'close' },
      ],
    },
    {
      label: menuText('help'),
      submenu: [
        {
          label: menuText('homepage'),
          click: () => { shell.openExternal('https://github.com/zy-eagle/envnexus'); },
        },
        { type: 'separator' },
        {
          label: `${menuText('about')}`,
          click: () => {
            dialog.showMessageBox({
              type: 'info',
              title: menuText('about'),
              message: `EnvNexus Agent\n${menuText('version')}: ${app.getVersion()}`,
            });
          },
        },
      ],
    },
  ];

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
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

function safeLocalAPI(method: string, path: string, body?: object, timeoutMs = 5000): Promise<any> {
  return localAPIRequest(method, path, body, timeoutMs).catch((err: Error) => ({
    error: err.message || 'agent-core not reachable',
  }));
}

function registerIPC(): void {
  ipcMain.handle('get-agent-status', () =>
    safeLocalAPI('GET', '/local/v1/runtime/status')
  );

  ipcMain.handle('export-diagnostics', () =>
    safeLocalAPI('POST', '/local/v1/diagnostics/export', {})
  );

  ipcMain.handle('send-diagnose', (_e, query: string, history: any[]) => {
    return new Promise((resolve) => {
      const postData = JSON.stringify({ intent: query, history });
      const req = http.request({
        hostname: '127.0.0.1',
        port: 17700,
        path: '/local/v1/diagnose',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(postData),
          'Accept': 'text/event-stream',
        },
      }, (res) => {
        let buffer = '';
        res.on('data', (chunk: Buffer) => {
          buffer += chunk.toString();
          const lines = buffer.split('\n');
          buffer = lines.pop() || '';
          let currentEvent = '';
          for (const line of lines) {
            if (line.startsWith('event: ')) {
              currentEvent = line.slice(7).trim();
            } else if (line.startsWith('data: ') && currentEvent) {
              try {
                const data = JSON.parse(line.slice(6));
                if (currentEvent === 'progress' && mainWindow) {
                  mainWindow.webContents.send('diagnosis-progress', data);
                } else if (currentEvent === 'result') {
                  resolve(data);
                } else if (currentEvent === 'error') {
                  resolve(data);
                }
              } catch { /* ignore parse errors */ }
              currentEvent = '';
            }
          }
        });
        res.on('end', () => {
          if (buffer.includes('data: ')) {
            const dataLine = buffer.split('\n').find(l => l.startsWith('data: '));
            if (dataLine) {
              try { resolve(JSON.parse(dataLine.slice(6))); } catch { /* ignore */ }
            }
          }
        });
      });
      req.on('error', (e) => resolve({ error: `agent-core not reachable: ${e.message}` }));
      req.setTimeout(660000, () => { req.destroy(); resolve({ error: 'diagnosis timed out (660s)' }); });
      req.write(postData);
      req.end();
    });
  });

  let activeChatRequest: http.ClientRequest | null = null;
  let activeChatSessionId: string | null = null;
  let chatCancelling = false;

  ipcMain.handle('send-chat', (_e, messages: Array<{ role: string; content: string }>) => {
    chatCancelling = false;
    return new Promise((resolve) => {
      let resolved = false;
      const safeResolve = (val: any) => {
        if (resolved) return;
        resolved = true;
        activeChatRequest = null;
        activeChatSessionId = null;
        resolve(val);
      };

      const settings = loadSettings();
      const chatBody: Record<string, unknown> = { messages };
      if (settings.maxIterations && settings.maxIterations > 0) {
        chatBody.max_iterations = settings.maxIterations;
      }
      const postData = JSON.stringify(chatBody);
      const req = http.request({
        hostname: '127.0.0.1',
        port: 17700,
        path: '/local/v1/chat',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Content-Length': Buffer.byteLength(postData),
          'Accept': 'text/event-stream',
        },
      }, (res) => {
        let buffer = '';
        res.on('data', (chunk: Buffer) => {
          buffer += chunk.toString();
          const lines = buffer.split('\n');
          buffer = lines.pop() || '';
          let currentEvent = '';
          for (const line of lines) {
            if (line.startsWith('event: ')) {
              currentEvent = line.slice(7).trim();
            } else if (line.startsWith('data: ') && currentEvent) {
              try {
                const data = JSON.parse(line.slice(6));
                if (currentEvent === 'session') {
                  activeChatSessionId = data.session_id || null;
                } else if (currentEvent === 'done') {
                  safeResolve(data);
                } else if (currentEvent === 'error') {
                  safeResolve(data);
                } else if (currentEvent === 'cancelled') {
                  safeResolve({ cancelled: true, message: data.message || 'Cancelled' });
                } else if (mainWindow) {
                  mainWindow.webContents.send('chat-event', { type: currentEvent, content: data });
                }
              } catch { /* ignore parse errors */ }
              currentEvent = '';
            }
          }
        });
        res.on('end', () => {
          if (chatCancelling) {
            safeResolve({ cancelled: true, message: 'Cancelled' });
            return;
          }
          if (buffer.includes('data: ')) {
            const dataLine = buffer.split('\n').find(l => l.startsWith('data: '));
            if (dataLine) {
              try { safeResolve(JSON.parse(dataLine.slice(6))); return; } catch { /* ignore */ }
            }
          }
          safeResolve({ error: 'Connection closed unexpectedly' });
        });
      });
      req.on('error', (e) => {
        if (chatCancelling) {
          safeResolve({ cancelled: true, message: 'Cancelled' });
        } else {
          safeResolve({ error: `agent-core not reachable: ${e.message}` });
        }
      });
      req.setTimeout(660000, () => {
        req.destroy();
        safeResolve({ error: 'chat timed out (660s)' });
      });
      activeChatRequest = req;
      req.write(postData);
      req.end();
    });
  });

  ipcMain.handle('cancel-chat', async () => {
    chatCancelling = true;
    if (activeChatSessionId) {
      try {
        await localAPIRequest('POST', '/local/v1/chat/cancel', { session_id: activeChatSessionId }, 3000);
      } catch { /* best effort */ }
    }
    setTimeout(() => {
      if (activeChatRequest) {
        activeChatRequest.destroy();
        activeChatRequest = null;
      }
      activeChatSessionId = null;
    }, 2000);
    return { ok: true };
  });

  ipcMain.handle('chat-approve', (_e, approvalId: string, approved: boolean) =>
    safeLocalAPI('POST', '/local/v1/chat/approve', { approval_id: approvalId, approved })
  );

  ipcMain.handle('chat-auto-approve', (_e, enabled: boolean) => {
    if (!activeChatSessionId) return { error: 'no active chat session' };
    return safeLocalAPI('POST', '/local/v1/chat/auto-approve', {
      session_id: activeChatSessionId,
      enabled,
    });
  });

  ipcMain.handle('get-settings', () => loadSettings());

  ipcMain.handle('save-settings', (_e, settings: Settings) => {
    saveSettings(settings);
    buildAppMenu();
    buildTrayMenu();
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

  // ── Agent-core self-update IPC ──
  ipcMain.handle('agent-update-status', () =>
    safeLocalAPI('GET', '/local/v1/update/status')
  );

  ipcMain.handle('agent-update-check', () =>
    safeLocalAPI('POST', '/local/v1/update/check')
  );

  ipcMain.handle('agent-update-download', () =>
    safeLocalAPI('POST', '/local/v1/update/download')
  );

  ipcMain.handle('agent-update-apply', async () => {
    const result = await safeLocalAPI('POST', '/local/v1/update/apply');
    if (result && !result.error) {
      stopAgentCore();
      lastSpawnTime = 0;
      // Windows: allow rename/AV hooks on the replaced enx-agent.exe to settle before spawn (avoids spawn UNKNOWN).
      const delayMs = process.platform === 'win32' ? 4500 : 2000;
      setTimeout(spawnAgentCore, delayMs);
    }
    return result;
  });

  // ── Desktop self-update IPC (portable + installer) ──
  ipcMain.handle('desktop-update-check', async () => {
    if (isPortableMode()) {
      const info = await checkGitHubRelease();
      if (!info) return { has_update: false };
      const current = app.getVersion();
      if (compareSemver(info.version, current) <= 0) return { has_update: false, current_version: current };
      portableUpdateInfo = info;
      return { has_update: true, current_version: current, latest_version: info.version };
    } else {
      try {
        const { autoUpdater } = require('electron-updater');
        await autoUpdater.checkForUpdates();
        return { has_update: false, message: 'check triggered (installer mode)' };
      } catch {
        return { has_update: false, message: 'electron-updater not available' };
      }
    }
  });

  ipcMain.handle('desktop-update-download', async () => {
    if (!isPortableMode() || !portableUpdateInfo) {
      return { error: 'no portable update available' };
    }
    try {
      const zipPath = await downloadPortableUpdate(portableUpdateInfo);
      portableUpdateInfo.zipPath = zipPath;
      if (mainWindow) {
        mainWindow.webContents.send('update-downloaded', { type: 'desktop', version: portableUpdateInfo.version });
      }
      return { status: 'downloaded', version: portableUpdateInfo.version, path: zipPath };
    } catch (err: any) {
      return { error: err.message || 'download failed' };
    }
  });

  ipcMain.handle('desktop-update-apply', () => {
    if (!isPortableMode() || !portableUpdateInfo?.zipPath) {
      // Installer mode: electron-updater handles this on quit
      try {
        const { autoUpdater } = require('electron-updater');
        autoUpdater.quitAndInstall(false, true);
      } catch {}
      return { status: 'restarting' };
    }
    stopAgentCore();
    applyPortableUpdate(portableUpdateInfo.zipPath);
    return { status: 'applying' };
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
  if (isPortableMode()) {
    initPortableAutoUpdate();
  } else {
    initInstallerAutoUpdate();
  }

  // Poll agent-core update status regardless of desktop distribution type
  setInterval(async () => {
    try {
      const result = await localAPIRequest('GET', '/local/v1/update/status');
      if (result && mainWindow) {
        mainWindow.webContents.send('agent-update-status', result);
      }
    } catch {}
  }, 60_000);
}

// NSIS installer: electron-updater handles everything
function initInstallerAutoUpdate(): void {
  try {
    const { autoUpdater } = require('electron-updater');

    autoUpdater.autoDownload = false;
    autoUpdater.autoInstallOnAppQuit = true;

    autoUpdater.on('update-available', (info: any) => {
      console.log(`[updater] Desktop update available: ${info.version}`);
      if (mainWindow) {
        mainWindow.webContents.send('update-available', { type: 'desktop', version: info.version });
      }
      autoUpdater.downloadUpdate();
    });

    autoUpdater.on('download-progress', (progress: any) => {
      if (mainWindow) {
        mainWindow.webContents.send('update-progress', { type: 'desktop', percent: Math.round(progress.percent) });
      }
    });

    autoUpdater.on('update-downloaded', (info: any) => {
      console.log(`[updater] Desktop update downloaded: ${info.version}`);
      if (mainWindow) {
        mainWindow.webContents.send('update-downloaded', { type: 'desktop', version: info.version });
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

// Portable (ZIP) edition: check GitHub releases API for newer version,
// download the ZIP to data/updates/, and self-extract on apply.
let portableUpdateInfo: { version: string; downloadUrl: string; zipPath?: string } | null = null;

function initPortableAutoUpdate(): void {
  const checkPortableUpdate = async () => {
    try {
      const info = await checkGitHubRelease();
      if (!info) return;

      const currentVer = app.getVersion();
      if (compareSemver(info.version, currentVer) <= 0) {
        console.log('[portable-updater] Already on latest version');
        return;
      }

      console.log(`[portable-updater] Update available: ${currentVer} -> ${info.version}`);
      portableUpdateInfo = info;
      if (mainWindow) {
        mainWindow.webContents.send('update-available', { type: 'desktop', version: info.version });
      }
    } catch (err) {
      console.error('[portable-updater] Check failed:', err);
    }
  };

  // First check after 15s, then every 4h
  setTimeout(checkPortableUpdate, 15_000);
  setInterval(checkPortableUpdate, 4 * 60 * 60 * 1000);
}

interface GHRelease {
  tag_name: string;
  assets: Array<{ name: string; browser_download_url: string }>;
}

function checkGitHubRelease(): Promise<{ version: string; downloadUrl: string } | null> {
  return new Promise((resolve) => {
    const options = {
      hostname: 'api.github.com',
      path: '/repos/zy-eagle/envnexus/releases/latest',
      headers: { 'User-Agent': `EnvNexus-Agent/${app.getVersion()}` },
    };

    https.get(options, (res) => {
      if (res.statusCode === 302 || res.statusCode === 301) {
        https.get(res.headers.location || '', (redirectRes) => {
          handleGHResponse(redirectRes, resolve);
        }).on('error', () => resolve(null));
        return;
      }
      handleGHResponse(res, resolve);
    }).on('error', () => resolve(null));
  });
}

function handleGHResponse(
  res: http.IncomingMessage,
  resolve: (val: { version: string; downloadUrl: string } | null) => void,
): void {
  let body = '';
  res.on('data', (chunk: Buffer) => { body += chunk.toString(); });
  res.on('end', () => {
    try {
      const release: GHRelease = JSON.parse(body);
      const version = release.tag_name.replace(/^v/, '');

      const platform = process.platform === 'win32' ? 'win' : process.platform === 'darwin' ? 'mac' : 'linux';
      const zipAsset = release.assets.find((a) =>
        a.name.toLowerCase().includes(platform) && a.name.toLowerCase().endsWith('.zip')
      );

      if (!zipAsset) {
        console.log('[portable-updater] No matching ZIP asset found in release');
        resolve(null);
        return;
      }

      resolve({ version, downloadUrl: zipAsset.browser_download_url });
    } catch {
      resolve(null);
    }
  });
  res.on('error', () => resolve(null));
}

function downloadPortableUpdate(info: { version: string; downloadUrl: string }): Promise<string> {
  return new Promise((resolve, reject) => {
    const updateDir = path.join(getAgentDataDir(), 'updates');
    fs.mkdirSync(updateDir, { recursive: true });
    const zipPath = path.join(updateDir, `envnexus-agent-${info.version}.zip`);

    // Skip if already downloaded
    if (fs.existsSync(zipPath)) {
      console.log('[portable-updater] ZIP already downloaded:', zipPath);
      resolve(zipPath);
      return;
    }

    const doDownload = (downloadUrl: string, redirectCount = 0) => {
      if (redirectCount > 5) {
        reject(new Error('Too many redirects'));
        return;
      }

      const getter = downloadUrl.startsWith('https') ? https : http;
      getter.get(downloadUrl, { headers: { 'User-Agent': `EnvNexus-Agent/${app.getVersion()}` } }, (res) => {
        if (res.statusCode === 302 || res.statusCode === 301) {
          doDownload(res.headers.location || '', redirectCount + 1);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`Download failed: HTTP ${res.statusCode}`));
          return;
        }

        const tmpPath = zipPath + '.tmp';
        const fileStream = fs.createWriteStream(tmpPath);
        const totalSize = parseInt(res.headers['content-length'] || '0', 10);
        let downloaded = 0;

        res.on('data', (chunk: Buffer) => {
          fileStream.write(chunk);
          downloaded += chunk.length;
          if (totalSize > 0 && mainWindow) {
            const percent = Math.round((downloaded / totalSize) * 100);
            mainWindow.webContents.send('update-progress', { type: 'desktop', percent });
          }
        });

        res.on('end', () => {
          fileStream.end(() => {
            fs.renameSync(tmpPath, zipPath);
            console.log('[portable-updater] Downloaded:', zipPath);
            resolve(zipPath);
          });
        });

        res.on('error', (err) => {
          fileStream.close();
          try { fs.unlinkSync(tmpPath); } catch {}
          reject(err);
        });
      }).on('error', reject);
    };

    doDownload(info.downloadUrl);
  });
}

function applyPortableUpdate(zipPath: string): void {
  const baseDir = getPortableBaseDir();
  const extractScript = path.join(baseDir, 'data', '_update.bat');

  // Write a self-update batch script that:
  // 1. Waits for the current process to exit
  // 2. Extracts the ZIP over the install directory
  // 3. Restarts the app
  // 4. Cleans up the script itself
  const exeName = path.basename(app.getPath('exe'));
  const script = `@echo off
echo Applying EnvNexus Agent update...
timeout /t 2 /nobreak >nul
powershell -Command "Expand-Archive -Path '${zipPath.replace(/'/g, "''")}' -DestinationPath '${baseDir.replace(/'/g, "''")}' -Force"
del "${zipPath}"
start "" "${path.join(baseDir, exeName)}"
del "%~f0"
`;

  fs.writeFileSync(extractScript, script, 'utf-8');
  child_process.spawn('cmd.exe', ['/c', extractScript], {
    detached: true,
    stdio: 'ignore',
    windowsHide: true,
  }).unref();

  app.quit();
}

function compareSemver(a: string, b: string): number {
  const pa = a.replace(/^v/, '').split('.').map(Number);
  const pb = b.replace(/^v/, '').split('.').map(Number);
  for (let i = 0; i < 3; i++) {
    const diff = (pa[i] || 0) - (pb[i] || 0);
    if (diff !== 0) return diff;
  }
  return 0;
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
  buildAppMenu();
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
