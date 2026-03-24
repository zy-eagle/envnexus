import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  // Agent Core status
  getAgentStatus: () => ipcRenderer.invoke('get-agent-status'),
  getConnectionStatus: () => ipcRenderer.invoke('get-connection-status'),
  getRecentSessions: () => ipcRenderer.invoke('get-recent-sessions'),

  // Approvals
  getPendingApprovals: () => ipcRenderer.invoke('get-pending-approvals'),
  resolveApproval: (id: string, approved: boolean) =>
    ipcRenderer.invoke('resolve-approval', id, approved),

  // Diagnostics
  exportDiagnostics: () => ipcRenderer.invoke('export-diagnostics'),

  // Settings
  getSettings: () => ipcRenderer.invoke('get-settings'),
  saveSettings: (settings: unknown) => ipcRenderer.invoke('save-settings', settings),

  // App
  getAppVersion: () => ipcRenderer.invoke('get-app-version'),
  openExternal: (url: string) => ipcRenderer.invoke('open-external', url),
  chooseAgentBinary: () => ipcRenderer.invoke('choose-agent-binary'),
  restartAgentCore: () => ipcRenderer.invoke('restart-agent-core'),
});
