import React, { useState, useEffect } from 'react';

interface APIKey {
  id: string;
  name: string;
  source_app: string;
  daily_quota: number;
  hourly_quota: number;
  active: boolean;
  created_at: string;
}

interface UsageStats {
  total_calls: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  average_latency_ms: number;
}

interface ModelItem {
  id: string;
  owned_by: string;
}

export default function App() {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [usage, setUsage] = useState<UsageStats>({
    total_calls: 0,
    total_prompt_tokens: 0,
    total_completion_tokens: 0,
    average_latency_ms: 0,
  });
  const [models, setModels] = useState<ModelItem[]>([]);
  
  // Key Generation state
  const [keyName, setKeyName] = useState('');
  const [sourceApp, setSourceApp] = useState('openwebui');
  const [dailyQuota, setDailyQuota] = useState(1000);
  const [hourlyQuota, setHourlyQuota] = useState(200);
  const [newlyCreatedKey, setNewlyCreatedKey] = useState<string | null>(null);

  // Status message state
  const [message, setMessage] = useState('');

  const refreshDashboard = async () => {
    try {
      const keysRes = await fetch('/api/admin/keys');
      if (keysRes.ok) {
        const keysData = await keysRes.json();
        setKeys(keysData || []);
      }

      const usageRes = await fetch('/api/admin/usage');
      if (usageRes.ok) {
        const usageData = await usageRes.json();
        setUsage(usageData || {
          total_calls: 0,
          total_prompt_tokens: 0,
          total_completion_tokens: 0,
          average_latency_ms: 0,
        });
      }

      const modelsRes = await fetch('/v1/models');
      if (modelsRes.ok) {
        const modelsData = await modelsRes.json();
        setModels(modelsData.data || []);
      }
    } catch (e) {
      console.error("Failed to load dashboard data:", e);
    }
  };

  useEffect(() => {
    refreshDashboard();
  }, []);

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    setNewlyCreatedKey(null);
    setMessage('');
    try {
      const res = await fetch('/api/admin/keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: keyName,
          source_app: sourceApp,
          daily_quota: dailyQuota,
          hourly_quota: hourlyQuota,
        }),
      });

      if (res.ok) {
        const data = await res.json();
        setNewlyCreatedKey(data.key);
        setMessage('API Key generated successfully! Please copy it now as it will not be shown again.');
        setKeyName('');
        refreshDashboard();
      } else {
        setMessage('Failed to generate key.');
      }
    } catch (e) {
      setMessage('Network error occurred.');
    }
  };

  return (
    <div style={{ fontFamily: 'system-ui, sans-serif', padding: '2rem', maxWidth: '1200px', margin: '0 auto' }}>
      <header style={{ borderBottom: '1px solid #334155', paddingBottom: '1rem', marginBottom: '2rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1 style={{ margin: 0, color: '#38bdf8' }}>NexusAI-Gateway</h1>
          <p style={{ margin: '4px 0 0 0', color: '#94a3b8' }}>High-Performance Go Control Plane</p>
        </div>
        <button onClick={refreshDashboard} style={{ background: '#0284c7', color: '#fff', border: 'none', padding: '0.5rem 1rem', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold' }}>
          Refresh Data
        </button>
      </header>

      {/* Grid Layout */}
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '2rem', marginBottom: '2rem' }}>
        
        {/* Telemetry Panel */}
        <section style={{ background: '#1e293b', padding: '1.5rem', borderRadius: '8px', border: '1px solid #334155' }}>
          <h2 style={{ margin: '0 0 1rem 0', color: '#f8fafc', fontSize: '1.2rem' }}>Live System Metrics</h2>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '1rem' }}>
            <div style={{ background: '#0f172a', padding: '1rem', borderRadius: '6px' }}>
              <span style={{ display: 'block', color: '#94a3b8', fontSize: '0.85rem' }}>Total LLM Requests</span>
              <strong style={{ fontSize: '1.5rem', color: '#38bdf8' }}>{usage.total_calls}</strong>
            </div>
            <div style={{ background: '#0f172a', padding: '1rem', borderRadius: '6px' }}>
              <span style={{ display: 'block', color: '#94a3b8', fontSize: '0.85rem' }}>Average Latency</span>
              <strong style={{ fontSize: '1.5rem', color: '#34d399' }}>{usage.average_latency_ms.toFixed(1)} ms</strong>
            </div>
            <div style={{ background: '#0f172a', padding: '1rem', borderRadius: '6px' }}>
              <span style={{ display: 'block', color: '#94a3b8', fontSize: '0.85rem' }}>Prompt Tokens Scubbed</span>
              <strong style={{ fontSize: '1.25rem', color: '#f43f5e' }}>{usage.total_prompt_tokens}</strong>
            </div>
            <div style={{ background: '#0f172a', padding: '1rem', borderRadius: '6px' }}>
              <span style={{ display: 'block', color: '#94a3b8', fontSize: '0.85rem' }}>Completion Tokens</span>
              <strong style={{ fontSize: '1.25rem', color: '#a78bfa' }}>{usage.total_completion_tokens}</strong>
            </div>
          </div>
        </section>

        {/* Dynamic Key Registration */}
        <section style={{ background: '#1e293b', padding: '1.5rem', borderRadius: '8px', border: '1px solid #334155' }}>
          <h2 style={{ margin: '0 0 1rem 0', color: '#f8fafc', fontSize: '1.2rem' }}>Create Secure Developer Key</h2>
          <form onSubmit={handleCreateKey} style={{ display: 'flex', flexDirection: 'column', gap: '0.8rem' }}>
            <div>
              <label style={{ display: 'block', color: '#94a3b8', fontSize: '0.85rem', marginBottom: '4px' }}>Key Label / Name</label>
              <input type="text" value={keyName} onChange={(e) => setKeyName(e.target.value)} required placeholder="e.g., Dev-Cluster-A" style={{ width: '100%', padding: '0.5rem', background: '#0f172a', border: '1px solid #334155', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: '1rem' }}>
              <div>
                <label style={{ display: 'block', color: '#94a3b8', fontSize: '0.85rem', marginBottom: '4px' }}>Source App</label>
                <select value={sourceApp} onChange={(e) => setSourceApp(e.target.value)} style={{ width: '100%', padding: '0.5rem', background: '#0f172a', border: '1px solid #334155', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }}>
                  <option value="openwebui">OpenWebUI</option>
                  <option value="openclaude">OpenClaude</option>
                  <option value="codex">Codex</option>
                  <option value="antigravity">Antigravity</option>
                </select>
              </div>
              <div>
                <label style={{ display: 'block', color: '#94a3b8', fontSize: '0.85rem', marginBottom: '4px' }}>Daily Quota</label>
                <input type="number" value={dailyQuota} onChange={(e) => setDailyQuota(parseInt(e.target.value))} style={{ width: '100%', padding: '0.5rem', background: '#0f172a', border: '1px solid #334155', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
              </div>
              <div>
                <label style={{ display: 'block', color: '#94a3b8', fontSize: '0.85rem', marginBottom: '4px' }}>Hourly Quota</label>
                <input type="number" value={hourlyQuota} onChange={(e) => setHourlyQuota(parseInt(e.target.value))} style={{ width: '100%', padding: '0.5rem', background: '#0f172a', border: '1px solid #334155', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
              </div>
            </div>
            <button type="submit" style={{ background: '#10b981', color: '#fff', border: 'none', padding: '0.6rem', borderRadius: '4px', cursor: 'pointer', fontWeight: 'bold', marginTop: '4px' }}>
              Generate API Token
            </button>
          </form>

          {message && <div style={{ marginTop: '1rem', padding: '0.75rem', borderRadius: '4px', background: '#0f172a', color: '#fbbf24', fontSize: '0.9rem', border: '1px solid #fbbf24' }}>{message}</div>}
          {newlyCreatedKey && (
            <div style={{ marginTop: '0.5rem', padding: '0.75rem', borderRadius: '4px', background: '#0284c7', color: '#fff', wordBreak: 'break-all', fontWeight: 'mono' }}>
              <strong>Your Token:</strong> <code>{newlyCreatedKey}</code>
            </div>
          )}
        </section>
      </div>

      {/* Keys List */}
      <section style={{ background: '#1e293b', padding: '1.5rem', borderRadius: '8px', border: '1px solid #334155', marginBottom: '2rem' }}>
        <h2 style={{ margin: '0 0 1rem 0', color: '#f8fafc', fontSize: '1.2rem' }}>Authorized Developer Keys</h2>
        <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid #334155', color: '#94a3b8', fontSize: '0.85rem' }}>
              <th style={{ padding: '0.75rem' }}>ID</th>
              <th style={{ padding: '0.75rem' }}>Label</th>
              <th style={{ padding: '0.75rem' }}>App Category</th>
              <th style={{ padding: '0.75rem' }}>Daily Quota</th>
              <th style={{ padding: '0.75rem' }}>Hourly Quota</th>
              <th style={{ padding: '0.75rem' }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {keys.length === 0 ? (
              <tr>
                <td colSpan={6} style={{ padding: '1.5rem', textAlign: 'center', color: '#64748b' }}>No API keys authorized yet.</td>
              </tr>
            ) : (
              keys.map((key) => (
                <tr key={key.id} style={{ borderBottom: '1px solid #334155', fontSize: '0.9rem' }}>
                  <td style={{ padding: '0.75rem', color: '#38bdf8' }}>{key.id}</td>
                  <td style={{ padding: '0.75rem', fontWeight: 'bold' }}>{key.name}</td>
                  <td style={{ padding: '0.75rem', color: '#a78bfa' }}>{key.source_app}</td>
                  <td style={{ padding: '0.75rem' }}>{key.daily_quota}</td>
                  <td style={{ padding: '0.75rem' }}>{key.hourly_quota}</td>
                  <td style={{ padding: '0.75rem' }}>
                    <span style={{ background: key.active ? '#065f46' : '#991b1b', color: key.active ? '#34d399' : '#f87171', padding: '0.2rem 0.5rem', borderRadius: '12px', fontSize: '0.75rem' }}>
                      {key.active ? 'Active' : 'Revoked'}
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </section>

      {/* Model Catalog */}
      <section style={{ background: '#1e293b', padding: '1.5rem', borderRadius: '8px', border: '1px solid #334155' }}>
        <h2 style={{ margin: '0 0 1rem 0', color: '#f8fafc', fontSize: '1.2rem' }}>Active Model Catalog</h2>
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(250px, 1fr))', gap: '1rem' }}>
          {models.map((m) => (
            <div key={m.id} style={{ background: '#0f172a', padding: '1rem', borderRadius: '6px', border: '1px solid #334155' }}>
              <span style={{ fontWeight: 'bold', color: '#38bdf8', display: 'block' }}>{m.id}</span>
              <span style={{ fontSize: '0.8rem', color: '#64748b' }}>Provider: {m.owned_by}</span>
            </div>
          ))}
        </div>
      </section>
    </div>
  );
}
