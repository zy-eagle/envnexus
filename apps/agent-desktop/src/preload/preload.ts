import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld('electronAPI', {
  getAgentStatus: () => ipcRenderer.invoke('get-agent-status')
});
