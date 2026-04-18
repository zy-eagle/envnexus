import { contextBridge, ipcRenderer } from 'electron';

const ipcChannels = [
  'diagnosis-progress',
  'chat-event',
  'agent-core-log',
  'update-available',
  'update-progress',
  'update-downloaded',
  'agent-update-status',
  'connection-status',
] as const;

function onChannel(channel: string, callback: (...args: any[]) => void) {
  const handler = (_event: Electron.IpcRendererEvent, ...args: any[]) => callback(...args);
  ipcRenderer.on(channel, handler);
}

contextBridge.exposeInMainWorld('electronAPI', {
  // Agent Core status
  getAgentStatus: () => ipcRenderer.invoke('get-agent-status'),
  getConnectionStatus: () => ipcRenderer.invoke('get-connection-status'),
  getRecentSessions: () => ipcRenderer.invoke('get-recent-sessions'),

  // Diagnostics
  exportDiagnostics: () => ipcRenderer.invoke('export-diagnostics'),
  sendDiagnose: (query: string, history: unknown[]) =>
    ipcRenderer.invoke('send-diagnose', query, history),
  onDiagnosisProgress: (callback: (data: { step: string; detail: string }) => void) => {
    onChannel('diagnosis-progress', callback);
  },

  // Chat (Agent Loop)
  sendChat: (messages: Array<{ role: string; content: unknown }>) =>
    ipcRenderer.invoke('send-chat', messages),
  cancelChat: () => ipcRenderer.invoke('cancel-chat'),
  chatApprove: (approvalId: string, approved: boolean) =>
    ipcRenderer.invoke('chat-approve', approvalId, approved),
  chatAutoApprove: (enabled: boolean) =>
    ipcRenderer.invoke('chat-auto-approve', enabled),
  onChatEvent: (callback: (event: { type: string; content: unknown }) => void) => {
    onChannel('chat-event', callback);
  },

  // Remediation Plan
  planApprove: (planId: string) =>
    ipcRenderer.invoke('plan-approve', planId),
  planReject: (planId: string) =>
    ipcRenderer.invoke('plan-reject', planId),
  planStepConfirm: (planId: string, stepId: number, approved: boolean) =>
    ipcRenderer.invoke('plan-step-confirm', planId, stepId, approved),
  planStepApprove: (planId: string, stepId: number, approved: boolean) =>
    ipcRenderer.invoke('plan-step-approve', planId, stepId, approved),

  // Watchlist
  watchlistCreate: (input: string) =>
    ipcRenderer.invoke('watchlist-create', input),
  watchlistConfirm: (items: unknown[]) =>
    ipcRenderer.invoke('watchlist-confirm', items),
  watchlistList: (source?: string) =>
    ipcRenderer.invoke('watchlist-list', source),
  watchlistGet: (id: string) =>
    ipcRenderer.invoke('watchlist-get', id),
  watchlistUpdate: (id: string, data: unknown) =>
    ipcRenderer.invoke('watchlist-update', id, data),
  watchlistDelete: (id: string) =>
    ipcRenderer.invoke('watchlist-delete', id),
  watchlistAlerts: (resolved?: string, limit?: number) =>
    ipcRenderer.invoke('watchlist-alerts', resolved, limit),
  healthScore: () =>
    ipcRenderer.invoke('health-score'),

  // Settings
  getSettings: () => ipcRenderer.invoke('get-settings'),
  saveSettings: (settings: unknown) => ipcRenderer.invoke('save-settings', settings),

  // App
  getAppVersion: () => ipcRenderer.invoke('get-app-version'),
  openExternal: (url: string) => ipcRenderer.invoke('open-external', url),
  chooseAgentBinary: () => ipcRenderer.invoke('choose-agent-binary'),
  restartAgentCore: () => ipcRenderer.invoke('restart-agent-core'),

  // Agent Core logs & detection
  getAgentCoreLogs: () => ipcRenderer.invoke('get-agent-core-logs'),
  getDetectedAgentPath: () => ipcRenderer.invoke('get-detected-agent-path'),
  onAgentCoreLog: (callback: (log: string) => void) => {
    onChannel('agent-core-log', callback);
  },

  // Self-update (agent-core)
  getAgentUpdateStatus: () => ipcRenderer.invoke('agent-update-status'),
  checkAgentUpdate: () => ipcRenderer.invoke('agent-update-check'),
  downloadAgentUpdate: () => ipcRenderer.invoke('agent-update-download'),
  applyAgentUpdate: () => ipcRenderer.invoke('agent-update-apply'),

  // Self-update (desktop app — portable + installer)
  checkDesktopUpdate: () => ipcRenderer.invoke('desktop-update-check'),
  downloadDesktopUpdate: () => ipcRenderer.invoke('desktop-update-download'),
  applyDesktopUpdate: () => ipcRenderer.invoke('desktop-update-apply'),

  // Connection status (pushed by main process health poller)
  onConnectionStatus: (callback: (status: string) => void) => {
    onChannel('connection-status', callback);
  },

  // Update events (shared)
  onUpdateAvailable: (callback: (data: { type: string; version: string }) => void) => {
    onChannel('update-available', callback);
  },
  onUpdateProgress: (callback: (data: { type: string; percent: number }) => void) => {
    onChannel('update-progress', callback);
  },
  onUpdateDownloaded: (callback: (data: { type: string; version: string }) => void) => {
    onChannel('update-downloaded', callback);
  },
  onAgentUpdateStatus: (callback: (status: any) => void) => {
    onChannel('agent-update-status', callback);
  },

  removeAllListeners: () => {
    for (const ch of ipcChannels) {
      ipcRenderer.removeAllListeners(ch);
    }
  },
});
