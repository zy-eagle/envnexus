"use client";

import { useState, useEffect, useCallback } from 'react';
import { useLanguage } from '@/lib/i18n/LanguageContext';
import { useDict } from '@/lib/i18n/dictionary';
import { api } from '@/lib/api/client';

interface DeviceGroup {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
}

interface GroupMember {
  id: string;
  device_group_id: string;
  device_id: string;
  created_at: string;
}

interface Device {
  id: string;
  device_name: string;
  hostname: string | null;
  platform: string;
  arch: string;
  status: string;
}

export default function DeviceGroupsPage({ params }: { params: { tenantId: string } }) {
  const { lang } = useLanguage();
  const t = useDict('deviceGroups', lang);
  const ct = useDict('common', lang);

  const [groups, setGroups] = useState<DeviceGroup[]>([]);
  const [loading, setLoading] = useState(true);
  const [showForm, setShowForm] = useState(false);
  const [formName, setFormName] = useState('');
  const [formDesc, setFormDesc] = useState('');
  const [creating, setCreating] = useState(false);

  const [expandedGroupId, setExpandedGroupId] = useState<string | null>(null);
  const [members, setMembers] = useState<GroupMember[]>([]);
  const [membersLoading, setMembersLoading] = useState(false);
  const [allDevices, setAllDevices] = useState<Device[]>([]);

  // Pagination states
  const [groupPagination, setGroupPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });
  const [memberPagination, setMemberPagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });
  const [devicePagination, setDevicePagination] = useState({
    page: 1,
    pageSize: 10,
    total: 0
  });

  const [showAddModal, setShowAddModal] = useState(false);
  const [selectedDeviceIds, setSelectedDeviceIds] = useState<Set<string>>(new Set());
  const [adding, setAdding] = useState(false);

  const fetchGroups = useCallback(async (page?: number, pageSize?: number) => {
    setLoading(true);
    try {
      const currentPage = page || groupPagination.page;
      const currentPageSize = pageSize || groupPagination.pageSize;
      const data = await api.get<any>(`/tenants/${params.tenantId}/device-groups?page=${currentPage}&page_size=${currentPageSize}`);
      setGroups(Array.isArray(data) ? data : (data?.items ?? []));
      setGroupPagination(prev => ({
        ...prev,
        page: currentPage,
        pageSize: currentPageSize,
        total: data?.total || 0
      }));
    } catch {
      setGroups([]);
    } finally {
      setLoading(false);
    }
  }, [params.tenantId, groupPagination.page, groupPagination.pageSize]);

  const fetchAllDevices = useCallback(async (page?: number, pageSize?: number) => {
    try {
      const currentPage = page || devicePagination.page;
      const currentPageSize = pageSize || devicePagination.pageSize;
      const data = await api.get<any>(`/tenants/${params.tenantId}/devices?page=${currentPage}&page_size=${currentPageSize}`);
      setAllDevices(Array.isArray(data) ? data : (data?.items ?? []));
      setDevicePagination(prev => ({
        ...prev,
        page: currentPage,
        pageSize: currentPageSize,
        total: data?.total || 0
      }));
    } catch {
      setAllDevices([]);
    }
  }, [params.tenantId, devicePagination.page, devicePagination.pageSize]);

  useEffect(() => {
    fetchGroups();
    fetchAllDevices();
  }, [fetchGroups, fetchAllDevices]);

  const fetchMembers = useCallback(async (groupId: string, page?: number, pageSize?: number) => {
    setMembersLoading(true);
    try {
      const currentPage = page || memberPagination.page;
      const currentPageSize = pageSize || memberPagination.pageSize;
      const data = await api.get<any>(`/tenants/${params.tenantId}/device-groups/${groupId}/members?page=${currentPage}&page_size=${currentPageSize}`);
      setMembers(Array.isArray(data) ? data : (data?.items ?? []));
      setMemberPagination(prev => ({
        ...prev,
        page: currentPage,
        pageSize: currentPageSize,
        total: data?.total || 0
      }));
    } catch {
      setMembers([]);
    } finally {
      setMembersLoading(false);
    }
  }, [params.tenantId, memberPagination.page, memberPagination.pageSize]);

  const toggleExpand = (groupId: string) => {
    if (expandedGroupId === groupId) {
      setExpandedGroupId(null);
      setMembers([]);
    } else {
      setExpandedGroupId(groupId);
      fetchMembers(groupId, 1, memberPagination.pageSize);
    }
  };

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formName) return;
    setCreating(true);
    try {
      await api.post(`/tenants/${params.tenantId}/device-groups`, { name: formName, description: formDesc });
      setFormName('');
      setFormDesc('');
      setShowForm(false);
      fetchGroups();
    } catch (error) {
      console.error('Failed to create device group:', error);
    } finally {
      setCreating(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!confirm(ct.confirmDelete)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/device-groups/${id}`);
      if (expandedGroupId === id) {
        setExpandedGroupId(null);
        setMembers([]);
      }
      fetchGroups();
    } catch (error) {
      console.error('Failed to delete device group:', error);
    }
  };

  const handleRemoveMember = async (deviceId: string) => {
    if (!expandedGroupId) return;
    if (!confirm((t as any).removeConfirm)) return;
    try {
      await api.delete(`/tenants/${params.tenantId}/device-groups/${expandedGroupId}/members/${deviceId}`);
      fetchMembers(expandedGroupId, memberPagination.page, memberPagination.pageSize);
    } catch (error) {
      console.error('Failed to remove member:', error);
    }
  };

  // Pagination handlers
  const handleGroupPageChange = (newPage: number) => {
    fetchGroups(newPage, groupPagination.pageSize);
  };

  const handleGroupPageSizeChange = (newPageSize: number) => {
    fetchGroups(1, newPageSize);
  };

  const handleMemberPageChange = (newPage: number) => {
    if (expandedGroupId) {
      fetchMembers(expandedGroupId, newPage, memberPagination.pageSize);
    }
  };

  const handleMemberPageSizeChange = (newPageSize: number) => {
    if (expandedGroupId) {
      fetchMembers(expandedGroupId, 1, newPageSize);
    }
  };

  const handleDevicePageChange = (newPage: number) => {
    fetchAllDevices(newPage, devicePagination.pageSize);
  };

  const handleDevicePageSizeChange = (newPageSize: number) => {
    fetchAllDevices(1, newPageSize);
  };

  const openAddModal = () => {
    setSelectedDeviceIds(new Set());
    setShowAddModal(true);
  };

  const handleAddDevices = async () => {
    if (!expandedGroupId || selectedDeviceIds.size === 0) return;
    setAdding(true);
    try {
      await api.post(`/tenants/${params.tenantId}/device-groups/${expandedGroupId}/members`, {
        device_ids: Array.from(selectedDeviceIds),
      });
      setShowAddModal(false);
      fetchMembers(expandedGroupId);
    } catch (error) {
      console.error('Failed to add members:', error);
    } finally {
      setAdding(false);
    }
  };

  const toggleDeviceSelection = (deviceId: string) => {
    setSelectedDeviceIds(prev => {
      const next = new Set(prev);
      if (next.has(deviceId)) next.delete(deviceId);
      else next.add(deviceId);
      return next;
    });
  };

  const memberDeviceIds = new Set(members.map(m => m.device_id));

  const getDeviceById = (deviceId: string) => allDevices.find(d => d.id === deviceId);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">{t.title}</h1>
        <button
          onClick={() => setShowForm(!showForm)}
          className="bg-indigo-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-indigo-700"
        >
          {ct.create}
        </button>
      </div>

      {showForm && (
        <form onSubmit={handleCreate} className="bg-white rounded-lg shadow-sm border border-gray-200 p-4 space-y-4">
          <input
            type="text"
            placeholder={ct.name}
            value={formName}
            onChange={e => setFormName(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
          />
          <input
            type="text"
            placeholder={t.description}
            value={formDesc}
            onChange={e => setFormDesc(e.target.value)}
            className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"
          />
          <div className="flex gap-2">
            <button type="submit" disabled={creating} className="bg-indigo-600 text-white px-4 py-2 rounded-md text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">{ct.save}</button>
            <button type="button" onClick={() => setShowForm(false)} className="border border-gray-300 px-4 py-2 rounded-md text-sm font-medium hover:bg-gray-50">{ct.cancel}</button>
          </div>
        </form>
      )}

      <div className="space-y-3">
        {loading ? (
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center text-gray-500">{ct.loading}</div>
        ) : groups.length === 0 ? (
          <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-8 text-center text-gray-500">{t.noGroups}</div>
        ) : (
          <>
            {groups.map(g => {
              const isExpanded = expandedGroupId === g.id;
              return (
                <div key={g.id} className="bg-white rounded-lg shadow-sm border border-gray-200 overflow-hidden">
                  {/* Group header row */}
                  <div
                    className={`flex items-center justify-between px-6 py-4 cursor-pointer hover:bg-gray-50 transition-colors ${isExpanded ? 'bg-indigo-50/50' : ''}`}
                    onClick={() => toggleExpand(g.id)}
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <svg
                        className={`w-4 h-4 text-gray-400 transition-transform duration-150 flex-shrink-0 ${isExpanded ? 'rotate-90' : ''}`}
                        fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor"
                      >
                        <path strokeLinecap="round" strokeLinejoin="round" d="m8.25 4.5 7.5 7.5-7.5 7.5" />
                      </svg>
                      <div className="min-w-0">
                        <div className="text-sm font-medium text-gray-900">{g.name}</div>
                        {g.description && <div className="text-xs text-gray-500 truncate">{g.description}</div>}
                      </div>
                    </div>
                    <div className="flex items-center gap-4 flex-shrink-0">
                      <span className="text-xs text-gray-400">{new Date(g.created_at).toLocaleDateString()}</span>
                      <button
                        onClick={e => { e.stopPropagation(); handleDelete(g.id); }}
                        className="text-red-500 hover:text-red-700 text-xs font-medium"
                      >
                        {ct.delete}
                      </button>
                    </div>
                  </div>

                  {/* Expanded members panel */}
                  {isExpanded && (
                    <div className="border-t border-gray-200">
                      <div className="px-6 py-3 bg-gray-50 flex items-center justify-between">
                        <span className="text-xs font-medium text-gray-600 uppercase tracking-wider">{(t as any).members}</span>
                        <button
                          onClick={openAddModal}
                          className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium text-indigo-700 bg-indigo-50 rounded-md hover:bg-indigo-100 transition-colors"
                        >
                          <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" d="M12 4.5v15m7.5-7.5h-15" />
                          </svg>
                          {(t as any).addDevices}
                        </button>
                      </div>

                      {membersLoading ? (
                        <div className="p-6 text-center text-gray-500 text-sm">{ct.loading}</div>
                      ) : members.length === 0 ? (
                        <div className="p-6 text-center text-gray-400 text-sm">{(t as any).noMembers}</div>
                      ) : (
                        <>
                          <table className="min-w-full divide-y divide-gray-200">
                            <thead className="bg-gray-50/50">
                              <tr>
                                <th className="px-6 py-2 text-left text-xs font-medium text-gray-500 uppercase">{(t as any).deviceName}</th>
                                <th className="px-6 py-2 text-left text-xs font-medium text-gray-500 uppercase">{(t as any).hostname}</th>
                                <th className="px-6 py-2 text-left text-xs font-medium text-gray-500 uppercase">{(t as any).platform}</th>
                                <th className="px-6 py-2 text-right text-xs font-medium text-gray-500 uppercase">{ct.actions}</th>
                              </tr>
                            </thead>
                            <tbody className="bg-white divide-y divide-gray-100">
                              {members.map(m => {
                                const dev = getDeviceById(m.device_id);
                                return (
                                  <tr key={m.id} className="hover:bg-gray-50">
                                    <td className="px-6 py-3 whitespace-nowrap text-sm text-gray-900">{dev?.device_name || m.device_id.substring(0, 12) + '...'}</td>
                                    <td className="px-6 py-3 whitespace-nowrap text-sm text-gray-500">{dev?.hostname || '-'}</td>
                                    <td className="px-6 py-3 whitespace-nowrap text-sm text-gray-500">
                                      {dev ? (
                                        <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-700">
                                          {dev.platform}/{dev.arch}
                                        </span>
                                      ) : '-'}
                                    </td>
                                    <td className="px-6 py-3 whitespace-nowrap text-right text-sm">
                                      <button
                                        onClick={() => handleRemoveMember(m.device_id)}
                                        className="text-red-500 hover:text-red-700 text-xs font-medium"
                                      >
                                        {(t as any).removeDevice}
                                      </button>
                                    </td>
                                  </tr>
                                );
                              })}
                            </tbody>
                          </table>
                          {/* Members pagination */}
                          <div className="flex justify-between items-center px-6 py-3 border-t border-gray-200 text-xs text-gray-500">
                            <div>共 {memberPagination.total} 条记录</div>
                            <div className="flex items-center space-x-2">
                              <span>每页显示：</span>
                              <select 
                                value={memberPagination.pageSize} 
                                onChange={(e) => handleMemberPageSizeChange(parseInt(e.target.value))}
                                className="border rounded-md px-1 py-0.5 text-xs"
                              >
                                <option value="10">10条</option>
                                <option value="20">20条</option>
                                <option value="50">50条</option>
                                <option value="100">100条</option>
                              </select>
                              <button 
                                onClick={() => handleMemberPageChange(memberPagination.page - 1)}
                                disabled={memberPagination.page === 1}
                                className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                              >
                                上一页
                              </button>
                              <span>{memberPagination.page}</span>
                              <button 
                                onClick={() => handleMemberPageChange(memberPagination.page + 1)}
                                disabled={memberPagination.page * memberPagination.pageSize >= memberPagination.total}
                                className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                              >
                                下一页
                              </button>
                            </div>
                          </div>
                        </>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
            {/* Groups pagination */}
            <div className="flex justify-between items-center px-6 py-4 border-t border-gray-200 text-xs text-gray-500">
              <div>共 {groupPagination.total} 个设备组</div>
              <div className="flex items-center space-x-2">
                <span>每页显示：</span>
                <select 
                  value={groupPagination.pageSize} 
                  onChange={(e) => handleGroupPageSizeChange(parseInt(e.target.value))}
                  className="border rounded-md px-1 py-0.5 text-xs"
                >
                  <option value="10">10条</option>
                  <option value="20">20条</option>
                  <option value="50">50条</option>
                  <option value="100">100条</option>
                </select>
                <button 
                  onClick={() => handleGroupPageChange(groupPagination.page - 1)}
                  disabled={groupPagination.page === 1}
                  className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                >
                  上一页
                </button>
                <span>{groupPagination.page}</span>
                <button 
                  onClick={() => handleGroupPageChange(groupPagination.page + 1)}
                  disabled={groupPagination.page * groupPagination.pageSize >= groupPagination.total}
                  className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                >
                  下一页
                </button>
              </div>
            </div>
          </>
        )}
      </div>

      {/* Add devices modal */}
      {showAddModal && (
        <>
          <div className="fixed inset-0 z-40 bg-black/30" onClick={() => setShowAddModal(false)} />
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div className="bg-white rounded-xl shadow-xl border border-gray-200 w-full max-w-lg max-h-[80vh] flex flex-col">
              <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
                <h3 className="text-base font-semibold text-gray-900">{(t as any).selectDevices}</h3>
                <button onClick={() => setShowAddModal(false)} className="text-gray-400 hover:text-gray-600">
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" strokeWidth={2} stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18 18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              <div className="flex-1 overflow-y-auto p-4">
                {allDevices.length === 0 ? (
                  <div className="text-center text-gray-400 py-8 text-sm">{(t as any).noAvailableDevices}</div>
                ) : (
                  <>
                    <div className="space-y-1">
                      {allDevices.map(dev => {
                        const inGroup = memberDeviceIds.has(dev.id);
                        const selected = selectedDeviceIds.has(dev.id);
                        return (
                          <label
                            key={dev.id}
                            className={`flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer transition-colors ${
                              inGroup ? 'opacity-50 cursor-not-allowed bg-gray-50' :
                              selected ? 'bg-indigo-50 ring-1 ring-indigo-200' : 'hover:bg-gray-50'
                            }`}
                          >
                            <input
                              type="checkbox"
                              checked={selected}
                              disabled={inGroup}
                              onChange={() => !inGroup && toggleDeviceSelection(dev.id)}
                              className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 disabled:opacity-50"
                            />
                            <div className="min-w-0 flex-1">
                              <div className="text-sm font-medium text-gray-900 truncate">
                                {dev.device_name || dev.id.substring(0, 12)}
                                {inGroup && <span className="ml-2 text-xs text-gray-400">({(t as any).alreadyInGroup})</span>}
                              </div>
                              <div className="text-xs text-gray-500 truncate">
                                {dev.hostname || '-'} &middot; {dev.platform}/{dev.arch}
                              </div>
                            </div>
                          </label>
                        );
                      })}
                    </div>
                    {/* Devices pagination */}
                    <div className="flex justify-between items-center px-3 py-3 border-t border-gray-200 text-xs text-gray-500 mt-4">
                      <div>共 {devicePagination.total} 台设备</div>
                      <div className="flex items-center space-x-2">
                        <span>每页显示：</span>
                        <select 
                          value={devicePagination.pageSize} 
                          onChange={(e) => handleDevicePageSizeChange(parseInt(e.target.value))}
                          className="border rounded-md px-1 py-0.5 text-xs"
                        >
                          <option value="10">10条</option>
                          <option value="20">20条</option>
                          <option value="50">50条</option>
                          <option value="100">100条</option>
                        </select>
                        <button 
                          onClick={() => handleDevicePageChange(devicePagination.page - 1)}
                          disabled={devicePagination.page === 1}
                          className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                        >
                          上一页
                        </button>
                        <span>{devicePagination.page}</span>
                        <button 
                          onClick={() => handleDevicePageChange(devicePagination.page + 1)}
                          disabled={devicePagination.page * devicePagination.pageSize >= devicePagination.total}
                          className="px-2 py-0.5 border rounded-md text-xs disabled:opacity-50"
                        >
                          下一页
                        </button>
                      </div>
                    </div>
                  </>
                )}
              </div>

              <div className="px-6 py-3 border-t border-gray-200 flex items-center justify-between bg-gray-50 rounded-b-xl">
                <span className="text-xs text-gray-500">
                  {selectedDeviceIds.size > 0 && `${selectedDeviceIds.size} ${(t as any).added || 'selected'}`}
                </span>
                <div className="flex gap-2">
                  <button
                    onClick={() => setShowAddModal(false)}
                    className="px-4 py-2 text-sm font-medium text-gray-700 border border-gray-300 rounded-md hover:bg-gray-50"
                  >
                    {ct.cancel}
                  </button>
                  <button
                    onClick={handleAddDevices}
                    disabled={selectedDeviceIds.size === 0 || adding}
                    className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 disabled:opacity-50"
                  >
                    {adding ? ct.loading : (t as any).addSelected}
                  </button>
                </div>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  );
}
