"use client";

import { useState } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';

const dict = {
  en: {
    title: "Download Packages",
    generateBtn: "Generate Link",
    modalTitle: "Generate Enrollment Link",
    modalDesc: "This feature will generate a signed, temporary download link for the EnvNexus Agent, pre-configured for this tenant.",
    cancel: "Cancel",
    generate: "Generate",
    noPackages: "No download packages generated yet for this tenant.",
    alertMsg: "Link generation is being implemented in the backend job-runner."
  },
  zh: {
    title: "下载包管理",
    generateBtn: "生成链接",
    modalTitle: "生成注册链接",
    modalDesc: "此功能将生成一个带签名的临时下载链接，EnvNexus Agent 下载后将自动绑定到当前租户。",
    cancel: "取消",
    generate: "生成",
    noPackages: "当前租户暂未生成任何下载包。",
    alertMsg: "链接生成功能正在后端 job-runner 中实现。"
  }
};

export default function DownloadPackagesPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = dict[lang];
  const [isModalOpen, setIsModalOpen] = useState(false);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button 
          onClick={() => setIsModalOpen(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-blue-700 transition-colors"
        >
          {t.generateBtn}
        </button>
      </div>

      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
            <h2 className="text-xl font-semibold mb-4">{t.modalTitle}</h2>
            <p className="text-sm text-gray-600 mb-6">
              {t.modalDesc}
            </p>
            <div className="flex justify-end space-x-3">
              <button 
                onClick={() => setIsModalOpen(false)}
                className="px-4 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                {t.cancel}
              </button>
              <button 
                onClick={() => {
                  alert(t.alertMsg);
                  setIsModalOpen(false);
                }}
                className="px-4 py-2 bg-blue-600 text-white rounded-md text-sm font-medium hover:bg-blue-700"
              >
                {t.generate}
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center text-gray-500">
        <p>{t.noPackages}</p>
      </div>
    </div>
  );
}
