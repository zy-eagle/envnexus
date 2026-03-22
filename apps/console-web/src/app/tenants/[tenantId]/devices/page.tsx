"use client";

import { useEffect, useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: {
    title: "Devices", addBtn: "Add Device", tenant: "Tenant", loading: "Loading devices...",
    deviceId: "Device ID", hostname: "Hostname", os: "OS", status: "Status", lastSeen: "Last Seen", actions: "Actions",
    diagnose: "Diagnose", modalTitle: "Add New Device",
    modalDesc: "To add a new device, you need to download and install the EnvNexus Agent on the target machine. Please go to the Download Packages page to generate an enrollment link for this tenant.",
    gotIt: "Got it"
  },
  zh: {
    title: "设备管理", addBtn: "添加设备", tenant: "所属租户", loading: "加载设备中...",
    deviceId: "设备 ID", hostname: "主机名", os: "操作系统", status: "状态", lastSeen: "最后在线", actions: "操作",
    diagnose: "诊断", modalTitle: "添加新设备",
    modalDesc: "要添加新设备，您需要在目标机器上下载并安装 EnvNexus Agent。请前往“下载包管理”页面生成此租户的注册链接。",
    gotIt: "知道了"
  }
};

export default function DevicesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = dict[lang];
  const [devices, setDevices] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);

  useEffect(() => {
    // Mock data for MVP
    setTimeout(() => {
      setDevices([
        { id: 'dev-1', hostname: 'win-workstation-01', status: 'online', os: 'windows', lastSeen: '2 mins ago' },
        { id: 'dev-2', hostname: 'mac-developer-02', status: 'offline', os: 'darwin', lastSeen: '5 hours ago' },
        { id: 'dev-3', hostname: 'linux-server-03', status: 'online', os: 'linux', lastSeen: 'Just now' },
      ]);
      setLoading(false);
    }, 500);
  }, [params.tenantId]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button 
          onClick={() => setIsModalOpen(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          {t.addBtn}
        </button>
      </div>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h2 className="text-xl font-semibold mb-4">{t.modalTitle}</h2>
            <p className="text-sm text-gray-600 mb-6">
              {t.modalDesc}
            </p>
            <div className="flex justify-end">
              <button 
                onClick={() => setIsModalOpen(false)}
                className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
              >
                {t.gotIt}
              </button>
            </div>
          </div>
        </div>
      )}
      
      <div className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
        <div className="px-6 py-4 border-b border-gray-200 bg-gray-50">
          <h2 className="text-sm font-medium text-gray-700">{t.tenant}: {params.tenantId}</h2>
        </div>
        
        {loading ? (
          <div className="p-8 text-center text-gray-500">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-4 border-gray-200 border-t-blue-600 mb-4"></div>
            <p>{t.loading}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.deviceId}
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.hostname}
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.os}
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.status}
                  </th>
                  <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.lastSeen}
                  </th>
                  <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    {t.actions}
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {devices.map((d: any) => (
                  <tr key={d.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      {d.id}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {d.hostname}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                        {d.os}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
                        d.status === 'online' 
                          ? 'bg-green-100 text-green-800' 
                          : 'bg-red-100 text-red-800'
                      }`}>
                        {d.status === 'online' ? (lang === 'zh' ? '在线' : 'online') : (lang === 'zh' ? '离线' : 'offline')}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {d.lastSeen}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <button className="text-blue-600 hover:text-blue-900">{t.diagnose}</button>
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
