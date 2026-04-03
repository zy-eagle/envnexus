import { contextBridge, ipcRenderer } from 'electron';

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
    ipcRenderer.on('diagnosis-progress', (_event, data) => callback(data));
  },

  // Chat (Agent Loop)
  sendChat: (messages: Array<{ role: string; content: string }>) =>
    ipcRenderer.invoke('send-chat', messages),
  cancelChat: () => ipcRenderer.invoke('cancel-chat'),
  chatApprove: (approvalId: string, approved: boolean) =>
    ipcRenderer.invoke('chat-approve', approvalId, approved),
  chatAutoApprove: (enabled: boolean) =>
    ipcRenderer.invoke('chat-auto-approve', enabled),
  onChatEvent: (callback: (event: { type: string; content: unknown }) => void) => {
    ipcRenderer.on('chat-event', (_event, data) => callback(data));
  },
  removeChatEventListeners: () => {
    ipcRenderer.removeAllListeners('chat-event');
  },

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
    ipcRenderer.on('agent-core-log', (_event, log: string) => callback(log));
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

  // Update events (shared)
  onUpdateAvailable: (callback: (data: { type: string; version: string }) => void) => {
    ipcRenderer.on('update-available', (_event, data) => callback(data));
  },
  onUpdateProgress: (callback: (data: { type: string; percent: number }) => void) => {
    ipcRenderer.on('update-progress', (_event, data) => callback(data));
  },
  onUpdateDownloaded: (callback: (data: { type: string; version: string }) => void) => {
    ipcRenderer.on('update-downloaded', (_event, data) => callback(data));
  },
  onAgentUpdateStatus: (callback: (status: any) => void) => {
    ipcRenderer.on('agent-update-status', (_event, status) => callback(status));
  },
});
