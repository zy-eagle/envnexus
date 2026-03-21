"use client";

import { useEffect, useState } from 'react';
import { apiClient } from '@/lib/api/client';

export default function DevicesPage({ params }: { params: { tenantId: string } }) {
  const [devices, setDevices] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // In a real app, we'd fetch from /api/v1/tenants/:id/devices
    // For MVP, we'll mock the data if the API isn't ready
    setDevices([
      { id: 'dev-1', hostname: 'win-workstation-01', status: 'online', os: 'windows' },
      { id: 'dev-2', hostname: 'mac-developer-02', status: 'offline', os: 'darwin' },
    ]);
    setLoading(false);
  }, [params.tenantId]);

  return (
    <div style={{ padding: '2rem', fontFamily: 'sans-serif' }}>
      <h1>Devices for Tenant: {params.tenantId}</h1>
      
      {loading ? (
        <p>Loading devices...</p>
      ) : (
        <table style={{ width: '100%', textAlign: 'left', borderCollapse: 'collapse', marginTop: '1rem' }}>
          <thead>
            <tr>
              <th style={{ borderBottom: '2px solid #ccc', padding: '10px' }}>Device ID</th>
              <th style={{ borderBottom: '2px solid #ccc', padding: '10px' }}>Hostname</th>
              <th style={{ borderBottom: '2px solid #ccc', padding: '10px' }}>OS</th>
              <th style={{ borderBottom: '2px solid #ccc', padding: '10px' }}>Status</th>
              <th style={{ borderBottom: '2px solid #ccc', padding: '10px' }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {devices.map((d: any) => (
              <tr key={d.id}>
                <td style={{ padding: '10px', borderBottom: '1px solid #eee' }}>{d.id}</td>
                <td style={{ padding: '10px', borderBottom: '1px solid #eee' }}>{d.hostname}</td>
                <td style={{ padding: '10px', borderBottom: '1px solid #eee' }}>{d.os}</td>
                <td style={{ padding: '10px', borderBottom: '1px solid #eee' }}>
                  <span style={{ 
                    color: d.status === 'online' ? 'green' : 'red',
                    fontWeight: 'bold'
                  }}>
                    {d.status}
                  </span>
                </td>
                <td style={{ padding: '10px', borderBottom: '1px solid #eee' }}>
                  <button style={{ padding: '5px 10px', cursor: 'pointer' }}>Diagnose</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
