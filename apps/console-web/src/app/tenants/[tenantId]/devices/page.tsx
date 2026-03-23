"use client";

import { useEffect, useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { api } from '@/lib/api/client';
import ConsoleLayout from '@/components/ConsoleLayout';

const dict = {
  en: {
    title: "Devices", addBtn: "Add Device", tenant: "Tenant", loading: "Loading devices...",
    deviceName: "Device Name", hostname: "Hostname", platform: "Platform", status: "Status", lastSeen: "Last Seen", actions: "Actions",
    version: "Agent Version", diagnose: "Diagnose", edit: "Edit", delete: "Delete", 
    modalTitle: "Add New Device", editModalTitle: "Edit Device",
    modalDesc: "To add a new device, download and install the EnvNexus Agent on the target machine. Go to Download Packages to generate an enrollment link.",
    gotIt: "Got it", confirmDelete: "Are you sure you want to delete this device?",
    cancel: "Cancel", save: "Save", noDevices: "No devices found."
  },
  zh: {
    title: "设备管理", addBtn: "添加设备", tenant: "所属租户", loading: "加载设备中...",
    deviceName: "设备名称", hostname: "主机名", platform: "平台", status: "状态", lastSeen: "最后在线", actions: "操作",
    version: "Agent 版本", diagnose: "诊断", edit: "编辑", delete: "删除",
    modalTitle: "添加新设备", editModalTitle: "编辑设备",
    modalDesc: "要添加新设备，请在目标机器上下载并安装 EnvNexus Agent。请前往下载包管理页面生成注册链接。",
    gotIt: "知道了", confirmDelete: "确定要删除此设备吗？",
    cancel: "取消", save: "保存", noDevices: "暂无设备。"
  }
};

function DevicesContent({ tenantId }: { tenantId: string }) {
  const { lang } = useLanguage();
  const t = dict[lang];
  const [devices, setDevices] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [isEditModalOpen, setIsEditModalOpen] = useState(false);
  const [editingDevice, setEditingDevice] = useState<any>(null);
  const [editName, setEditName] = useState('');
  const [editStatus, setEditStatus] = useState('');

  const fetchDevices = async () => {
    try {
      const data = await api.get<{ items: any[] }>(`/tenants/${tenantId}/devices`);
      setDevices(data.items || []);
    } catch (error) {
      console.error('Failed to fetch devices:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm(t.confirmDelete)) return;
    try {
      await api.delete(`/tenants/${tenantId}/devices/${id}`);
      fetchDevices();
    } catch (error) {
      console.error('Error deleting device:', error);
    }
  };

  const handleEditSave = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.put(`/tenants/${tenantId}/devices/${editingDevice.id}`, {
        device_name: editName,
        status: editStatus,
      });
      setIsEditModalOpen(false);
      fetchDevices();
    } catch (error) {
      console.error('Error updating device:', error);
    }
  };

  useEffect(() => { fetchDevices(); }, [tenantId]);

  const statusColor = (status: string) => {
    switch (status) {
      case 'active': return 'bg-green-100 text-green-800';
      case 'pending_activation': return 'bg-yellow-100 text-yellow-800';
      case 'quarantined': return 'bg-orange-100 text-orange-800';
      case 'revoked': case 'retired': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button onClick={() => setIsModalOpen(true)} className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700">
          {t.addBtn}
        </button>
      </div>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h2 className="text-xl font-semibold mb-4">{t.modalTitle}</h2>
            <p className="text-sm text-gray-600 mb-6">{t.modalDesc}</p>
            <div className="flex justify-end">
              <button onClick={() => setIsModalOpen(false)} className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700">{t.gotIt}</button>
            </div>
          </div>
        </div>
      )}

      {isEditModalOpen && editingDevice && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h2 className="text-xl font-semibold mb-4">{t.editModalTitle}</h2>
            <form onSubmit={handleEditSave} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.deviceName}</label>
                <input type="text" required value={editName} onChange={e => setEditName(e.target.value)}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm" />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">{t.status}</label>
                <select value={editStatus} onChange={e => setEditStatus(e.target.value)}
                  className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm">
                  <option value="active">Active</option>
                  <option value="pending_activation">Pending Activation</option>
                  <option value="quarantined">Quarantined</option>
                  <option value="revoked">Revoked</option>
                </select>
              </div>
              <div className="flex justify-end space-x-3 mt-6">
                <button type="button" onClick={() => setIsEditModalOpen(false)} className="px-4 py-2 border border-gray-300 text-gray-700 rounded-md text-sm">{t.cancel}</button>
                <button type="submit" className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700">{t.save}</button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-blue-600 mb-4"></div>
            <p>{t.loading}</p>
          </div>
        ) : devices.length === 0 ? (
          <div className="p-8 text-center text-gray-500">{t.noDevices}</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.deviceName}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.hostname}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.platform}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.version}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.status}</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase">{t.lastSeen}</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase">{t.actions}</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {devices.map((d: any) => (
                  <tr key={d.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{d.device_name}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{d.hostname || '-'}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                        {d.platform}/{d.arch}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{d.agent_version || '-'}</td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${statusColor(d.status)}`}>
                        {d.status}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {d.last_seen_at ? new Date(d.last_seen_at).toLocaleString() : 'Never'}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button className="text-blue-600 hover:text-blue-900 mr-3">{t.diagnose}</button>
                      <button onClick={() => {
                        setEditingDevice(d);
                        setEditName(d.device_name);
                        setEditStatus(d.status);
                        setIsEditModalOpen(true);
                      }} className="text-gray-600 hover:text-gray-900 mr-3">{t.edit}</button>
                      <button onClick={() => handleDelete(d.id)} className="text-red-600 hover:text-red-900">{t.delete}</button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

export default function DevicesPage({ params }: { params: { tenantId: string } }) {
  return (
    <ConsoleLayout>
      <DevicesContent tenantId={params.tenantId} />
    </ConsoleLayout>
  );
}
