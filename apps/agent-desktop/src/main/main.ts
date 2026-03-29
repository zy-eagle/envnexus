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

const DEFAULT_SETTINGS: Settings = {
  language: 'zh',
  platformURL: 'http://localhost:8080',
  logLevel: 'info',
  agentCorePath: '',
  autoStart: true,
};

const SETTINGS_FILE = path.join(app.getPath('userData'), 'settings.json');

// Agent data directory: aligned with the app's install location.
// On Windows (NSIS install): C:\Program Files\EnvNexus Agent\data
// Fallback: ~/.envnexus/agent
function getAgentDataDir(): string {
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

    req.on('error', (e) => reject({ error: 'agent-core not reachable', details: e.message }));
    req.setTimeout(5000, () => {
      req.destroy();
      reject({ error: 'timeout', details: 'agent-core did not respond in 5s' });
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

  const settings = loadSettings();
  const binaryPath = settings.agentCorePath || findAgentCoreBinary();

  if (!binaryPath || !fs.existsSync(binaryPath)) {
    console.warn('agent-core binary not found, skipping spawn');
    return;
  }

  console.log('Spawning agent-core:', binaryPath);

  const agentEnv: Record<string, string> = {
    ...process.env as Record<string, string>,
    ENX_LOG_LEVEL: settings.logLevel,
  };
  if (settings.platformURL) {
    agentEnv.ENX_PLATFORM_URL = settings.platformURL;
  }

  const agentDataDir = getAgentDataDir();
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
    console.log('[agent-core]', data.toString().trim());
  });

  agentCoreProcess.stderr?.on('data', (data) => {
    console.error('[agent-core]', data.toString().trim());
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

  const candidates = [
    // Electron packaged app: extraResources/bin/
    path.join(process.resourcesPath || '', 'bin', binaryName),
    // Development: project root bin/
    path.join(app.getAppPath(), '..', '..', 'bin', binaryName),
    // Same directory as the desktop app
    path.join(path.dirname(app.getPath('exe')), binaryName),
    // Current working directory
    path.join(process.cwd(), binaryName),
    path.join(process.cwd(), 'bin', binaryName),
  ];

  if (!isWin) {
    candidates.push('/usr/local/bin/enx-agent');
    candidates.push(path.join(process.env.HOME || '', '.local', 'bin', 'enx-agent'));
  }

  return candidates.find((p) => fs.existsSync(p)) || '';
}

function stopAgentCore(): void {
  if (agentCoreProcess) {
    agentCoreProcess.kill('SIGTERM');
    agentCoreProcess = null;
  }
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

  // Windows doesn't render SVG via createFromDataURL reliably.
  // Use a 32x32 PNG data URL built from raw RGBA pixel buffer.
  const size = 32;
  const buf = Buffer.alloc(size * size * 4, 0);

  const brandR = 99, brandG = 102, brandB = 241; // indigo-500 #6366f1
  const [dotR, dotG, dotB] = hexToRGB(dotColor);

  const cx = size / 2, cy = size / 2;
  for (let y = 0; y < size; y++) {
    for (let x = 0; x < size; x++) {
      const dx = x - cx + 0.5, dy = y - cy + 0.5;
      const dist = Math.sqrt(dx * dx + dy * dy);
      const idx = (y * size + x) * 4;

      if (dist <= 12) {
        buf[idx] = brandR; buf[idx + 1] = brandG; buf[idx + 2] = brandB; buf[idx + 3] = 255;
        // "E" letter — simplified pixel art in center
        const lx = x - 9, ly = y - 9;
        if (lx >= 0 && lx < 14 && ly >= 0 && ly < 14) {
          const isE = (lx >= 3 && lx <= 4) || // left bar
                      (ly >= 1 && ly <= 2 && lx >= 3 && lx <= 10) || // top bar
                      (ly >= 6 && ly <= 7 && lx >= 3 && lx <= 9) ||  // middle bar
                      (ly >= 11 && ly <= 12 && lx >= 3 && lx <= 10); // bottom bar
          if (isE) {
            buf[idx] = 255; buf[idx + 1] = 255; buf[idx + 2] = 255; buf[idx + 3] = 255;
          }
        }
      }

      // Status dot (bottom-right)
      const ddx = x - 24.5, ddy = y - 24.5;
      const dotDist = Math.sqrt(ddx * ddx + ddy * ddy);
      if (dotDist <= 5) {
        buf[idx] = dotR; buf[idx + 1] = dotG; buf[idx + 2] = dotB; buf[idx + 3] = 255;
      } else if (dotDist <= 6.5) {
        // White ring around dot
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
        setTimeout(spawnAgentCore, 500);
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

function createWindow(): void {
  mainWindow = new BrowserWindow({
    width: 960,
    height: 680,
    minWidth: 720,
    minHeight: 500,
    title: 'EnvNexus Agent',
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

function registerIPC(): void {
  ipcMain.handle('get-agent-status', () =>
    localAPIRequest('GET', '/local/v1/runtime/status')
  );

  ipcMain.handle('get-pending-approvals', () =>
    localAPIRequest('GET', '/local/v1/approvals/pending')
  );

  ipcMain.handle('resolve-approval', (_e, id: string, approved: boolean) =>
    localAPIRequest('POST', `/local/v1/approvals/${id}/resolve`, { approved })
  );

  ipcMain.handle('export-diagnostics', () =>
    localAPIRequest('POST', '/local/v1/diagnostics/export', {})
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
    setTimeout(spawnAgentCore, 500);
    return { ok: true };
  });

  ipcMain.handle('get-app-version', () => app.getVersion());

  ipcMain.handle('get-recent-sessions', () =>
    localAPIRequest('GET', '/local/v1/sessions/recent')
  );
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

  if (fs.existsSync(enxTarget) || fs.existsSync(jsonTarget)) return;

  // Search for .enx (TOML) config first, then legacy JSON
  const enxSearchPaths = [
    path.join(path.dirname(app.getPath('exe')), 'agent.enx'),
    path.join(process.resourcesPath || '', 'agent.enx'),
    path.join(app.getAppPath(), 'agent.enx'),
    path.join(app.getPath('downloads'), 'agent.enx'),
  ];

  for (const src of enxSearchPaths) {
    try {
      if (fs.existsSync(src)) {
        fs.mkdirSync(agentDataDir, { recursive: true });
        fs.copyFileSync(src, enxTarget);
        console.log('[config] Imported bundled agent.enx from:', src);

        const cfg = parseTOMLConfig(fs.readFileSync(src, 'utf-8'));
        if (cfg.platform_url) {
          const settings = loadSettings();
          settings.platformURL = cfg.platform_url;
          saveSettings(settings);
          console.log('[config] Updated desktop settings with platform URL:', cfg.platform_url);
        }
        return;
      }
    } catch (e) {
      console.warn('[config] Failed to import .enx from', src, e);
    }
  }

  // Fallback: legacy JSON config
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

// ── App lifecycle ──────────────────────────────────────────────────────────────

app.whenReady().then(() => {
  importBundledConfig();
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
