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

interface ProviderConn {
  id: string;
  provider: string;
  name: string;
  api_key: string;
  endpoint: string;
  is_active: boolean;
  priority: number;
}

interface UsageLog {
  id: number;
  key_id: string;
  model_id: string;
  prompt_tokens: number;
  completion_tokens: number;
  latency_ms: number;
  source_app: string;
  created_at: string;
}

interface ModelItem {
  id: string;
  name: string;
  provider: string;
}

interface UsageStats {
  total_calls: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  average_latency_ms: number;
}

export default function App() {
  const [activeTab, setActiveTab] = useState<'keys' | 'providers' | 'logs' | 'catalog'>('keys');

  // Core Data States
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [providers, setProviders] = useState<ProviderConn[]>([]);
  const [logs, setLogs] = useState<UsageLog[]>([]);
  const [models, setModels] = useState<ModelItem[]>([]);
  const [usage, setUsage] = useState<UsageStats>({
    total_calls: 0,
    total_prompt_tokens: 0,
    total_completion_tokens: 0,
    average_latency_ms: 0,
  });

  // Action / Creation States
  const [keyName, setKeyName] = useState('');
  const [sourceApp, setSourceApp] = useState('openwebui');
  const [dailyQuota, setDailyQuota] = useState(1000);
  const [hourlyQuota, setHourlyQuota] = useState(200);
  const [newlyCreatedKey, setNewlyCreatedKey] = useState<string | null>(null);

  const [providerId, setProviderId] = useState('openai');
  const [providerName, setProviderName] = useState('');
  const [providerKey, setProviderKey] = useState('');
  const [providerEndpoint, setProviderEndpoint] = useState('');
  const [providerPriority, setProviderPriority] = useState(1);

  const [message, setMessage] = useState('');
  const [sysVersion, setSysVersion] = useState({ version: '1.0.0', engine: 'Golang Engine' });

  const loadData = async () => {
    try {
      // 1. Fetch Keys
      const keysRes = await fetch('/api/admin/keys');
      if (keysRes.ok) {
        const keysData = await keysRes.json();
        setKeys(keysData || []);
      }

      // 2. Fetch Providers (OmniRoute Compatibility Endpoint)
      const provRes = await fetch('/api/providers');
      if (provRes.ok) {
        const provData = await provRes.json();
        setProviders(provData.connections || []);
      }

      // 3. Fetch Real-time logs
      const logsRes = await fetch('/api/admin/usage'); // fallback
      const detailLogsRes = await fetch('/api/admin/logs');
      if (detailLogsRes.ok) {
        const logsData = await detailLogsRes.json();
        setLogs(logsData || []);
      }
      if (logsRes.ok) {
        const usageData = await logsRes.json();
        setUsage(usageData || {
          total_calls: 0,
          total_prompt_tokens: 0,
          total_completion_tokens: 0,
          average_latency_ms: 0,
        });
      }

      // 4. Fetch models catalog (OmniRoute Compatibility Endpoint)
      const modelsRes = await fetch('/api/models');
      if (modelsRes.ok) {
        const modelsData = await modelsRes.json();
        setModels(modelsData.models || []);
      }

      const versionRes = await fetch('/api/system/version');
      if (versionRes.ok) {
        const versionData = await versionRes.json();
        setSysVersion(versionData);
      }
    } catch (e) {
      console.error('Failed to reload admin telemetry:', e);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
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
        setMessage('API Key generated successfully! Please copy it now.');
        setKeyName('');
        loadData();
      } else {
        setMessage('Failed to generate key.');
      }
    } catch (e) {
      setMessage('Network error occurred.');
    }
  };

  const handleAddProvider = async (e: React.FormEvent) => {
    e.preventDefault();
    setMessage('');
    try {
      const res = await fetch('/api/providers', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: providerId,
          provider: providerId,
          name: providerName || `${providerId} Connection`,
          api_key: providerKey,
          endpoint: providerEndpoint,
          is_active: true,
          priority: providerPriority,
        }),
      });

      if (res.ok) {
        setMessage(`Provider connection ${providerId} updated successfully!`);
        setProviderName('');
        setProviderKey('');
        setProviderEndpoint('');
        loadData();
      } else {
        setMessage('Failed to update provider.');
      }
    } catch (e) {
      setMessage('Network error.');
    }
  };

  return (
    <div style={{ minHeight: '100vh', background: '#0b0f19', color: '#f8fafc', fontFamily: 'system-ui, -apple-system, sans-serif' }}>
      {/* Top Banner */}
      <nav style={{ background: '#111827', borderBottom: '1px solid #1f2937', padding: '1rem 2rem', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{ background: '#38bdf8', width: '32px', height: '32px', borderRadius: '6px', display: 'flex', alignItems: 'center', justifyContent: 'center', fontWeight: 'bold', color: '#0f172a' }}>NX</div>
          <div>
            <h1 style={{ margin: 0, fontSize: '1.25rem', fontWeight: 800, color: '#38bdf8', letterSpacing: '0.5px' }}>NexusAI-Gateway</h1>
            <p style={{ margin: 0, fontSize: '0.75rem', color: '#94a3b8' }}>Go Microservice v{sysVersion.version} | {sysVersion.engine}</p>
          </div>
        </div>
        <div style={{ display: 'flex', gap: '8px' }}>
          <button onClick={() => setActiveTab('keys')} style={{ background: activeTab === 'keys' ? '#1e293b' : 'transparent', border: 'none', color: activeTab === 'keys' ? '#38bdf8' : '#94a3b8', padding: '0.5rem 1rem', borderRadius: '6px', cursor: 'pointer', fontWeight: 'bold' }}>API Keys</button>
          <button onClick={() => setActiveTab('providers')} style={{ background: activeTab === 'providers' ? '#1e293b' : 'transparent', border: 'none', color: activeTab === 'providers' ? '#38bdf8' : '#94a3b8', padding: '0.5rem 1rem', borderRadius: '6px', cursor: 'pointer', fontWeight: 'bold' }}>Providers Config</button>
          <button onClick={() => setActiveTab('logs')} style={{ background: activeTab === 'logs' ? '#1e293b' : 'transparent', border: 'none', color: activeTab === 'logs' ? '#38bdf8' : '#94a3b8', padding: '0.5rem 1rem', borderRadius: '6px', cursor: 'pointer', fontWeight: 'bold' }}>Request Logs</button>
          <button onClick={() => setActiveTab('catalog')} style={{ background: activeTab === 'catalog' ? '#1e293b' : 'transparent', border: 'none', color: activeTab === 'catalog' ? '#38bdf8' : '#94a3b8', padding: '0.5rem 1rem', borderRadius: '6px', cursor: 'pointer', fontWeight: 'bold' }}>Model Catalog</button>
        </div>
      </nav>

      {/* Main Body */}
      <div style={{ padding: '2rem', maxWidth: '1400px', margin: '0 auto' }}>
        
        {/* Core Stats Overview */}
        <section style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '1.5rem', marginBottom: '2rem' }}>
          <div style={{ background: '#111827', border: '1px solid #1f2937', padding: '1.25rem', borderRadius: '8px' }}>
            <span style={{ display: 'block', color: '#94a3b8', fontSize: '0.8rem', fontWeight: 600 }}>Total Requests</span>
            <strong style={{ fontSize: '1.75rem', color: '#38bdf8', display: 'block', marginTop: '4px' }}>{usage.total_calls}</strong>
          </div>
          <div style={{ background: '#111827', border: '1px solid #1f2937', padding: '1.25rem', borderRadius: '8px' }}>
            <span style={{ display: 'block', color: '#94a3b8', fontSize: '0.8rem', fontWeight: 600 }}>Average Latency</span>
            <strong style={{ fontSize: '1.75rem', color: '#10b981', display: 'block', marginTop: '4px' }}>{usage.average_latency_ms.toFixed(0)} ms</strong>
          </div>
          <div style={{ background: '#111827', border: '1px solid #1f2937', padding: '1.25rem', borderRadius: '8px' }}>
            <span style={{ display: 'block', color: '#94a3b8', fontSize: '0.8rem', fontWeight: 600 }}>Prompt Tokens Scubbed</span>
            <strong style={{ fontSize: '1.75rem', color: '#f43f5e', display: 'block', marginTop: '4px' }}>{usage.total_prompt_tokens}</strong>
          </div>
          <div style={{ background: '#111827', border: '1px solid #1f2937', padding: '1.25rem', borderRadius: '8px' }}>
            <span style={{ display: 'block', color: '#94a3b8', fontSize: '0.8rem', fontWeight: 600 }}>Completion Output</span>
            <strong style={{ fontSize: '1.75rem', color: '#8b5cf6', display: 'block', marginTop: '4px' }}>{usage.total_completion_tokens}</strong>
          </div>
        </section>

        {message && (
          <div style={{ background: '#1e1b4b', border: '1px solid #4338ca', color: '#fbbf24', padding: '1rem', borderRadius: '6px', marginBottom: '2rem', display: 'flex', justifyContent: 'space-between' }}>
            <span>{message}</span>
            <button onClick={() => setMessage('')} style={{ background: 'transparent', border: 'none', color: '#fff', cursor: 'pointer' }}>✕</button>
          </div>
        )}

        {/* Tab 1: API Keys Panel */}
        {activeTab === 'keys' && (
          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '2rem' }}>
            {/* Table */}
            <div style={{ background: '#111827', border: '1px solid #1f2937', borderRadius: '8px', padding: '1.5rem' }}>
              <h3 style={{ margin: '0 0 1rem 0', color: '#f8fafc' }}>Authorized Client Keys</h3>
              <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
                <thead>
                  <tr style={{ borderBottom: '1px solid #1f2937', color: '#94a3b8', fontSize: '0.8rem' }}>
                    <th style={{ padding: '0.75rem' }}>ID</th>
                    <th style={{ padding: '0.75rem' }}>Label Name</th>
                    <th style={{ padding: '0.75rem' }}>App Source</th>
                    <th style={{ padding: '0.75rem' }}>Daily Quota</th>
                    <th style={{ padding: '0.75rem' }}>Hourly Quota</th>
                    <th style={{ padding: '0.75rem' }}>Status</th>
                  </tr>
                </thead>
                <tbody>
                  {keys.length === 0 ? (
                    <tr><td colSpan={6} style={{ padding: '2rem', textAlign: 'center', color: '#64748b' }}>No keys active. Generated tokens will be displayed here.</td></tr>
                  ) : (
                    keys.map((k) => (
                      <tr key={k.id} style={{ borderBottom: '1px solid #1f2937', fontSize: '0.875rem' }}>
                        <td style={{ padding: '0.75rem', color: '#38bdf8', fontFamily: 'monospace' }}>{k.id}</td>
                        <td style={{ padding: '0.75rem', fontWeight: 'bold' }}>{k.name}</td>
                        <td style={{ padding: '0.75rem', color: '#a78bfa' }}>{k.source_app}</td>
                        <td style={{ padding: '0.75rem' }}>{k.daily_quota} req</td>
                        <td style={{ padding: '0.75rem' }}>{k.hourly_quota} req</td>
                        <td style={{ padding: '0.75rem' }}>
                          <span style={{ padding: '2px 8px', borderRadius: '12px', background: k.active ? '#065f46' : '#991b1b', color: k.active ? '#34d399' : '#f87171', fontSize: '0.75rem', fontWeight: 'bold' }}>
                            {k.active ? 'ACTIVE' : 'REVOKED'}
                          </span>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>

            {/* Create form */}
            <div style={{ background: '#111827', border: '1px solid #1f2937', borderRadius: '8px', padding: '1.5rem', height: 'fit-content' }}>
              <h3 style={{ margin: '0 0 1rem 0', color: '#f8fafc' }}>Create Client Token</h3>
              <form onSubmit={handleCreateKey} style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                <div>
                  <label style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginBottom: '6px' }}>Key Description</label>
                  <input type="text" value={keyName} onChange={(e) => setKeyName(e.target.value)} required placeholder="e.g. Cursor-Copilot-A" style={{ width: '100%', padding: '0.5rem', background: '#0b0f19', border: '1px solid #1f2937', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginBottom: '6px' }}>Source App Category</label>
                  <select value={sourceApp} onChange={(e) => setSourceApp(e.target.value)} style={{ width: '100%', padding: '0.5rem', background: '#0b0f19', border: '1px solid #1f2937', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }}>
                    <option value="openwebui">OpenWebUI</option>
                    <option value="openclaude">OpenClaude</option>
                    <option value="codex">Codex</option>
                    <option value="antigravity">Antigravity</option>
                    <option value="direct-api">Direct API</option>
                  </select>
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginBottom: '6px' }}>Daily Quota</label>
                  <input type="number" value={dailyQuota} onChange={(e) => setDailyQuota(parseInt(e.target.value))} style={{ width: '100%', padding: '0.5rem', background: '#0b0f19', border: '1px solid #1f2937', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginBottom: '6px' }}>Hourly Quota</label>
                  <input type="number" value={hourlyQuota} onChange={(e) => setHourlyQuota(parseInt(e.target.value))} style={{ width: '100%', padding: '0.5rem', background: '#0b0f19', border: '1px solid #1f2937', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
                </div>
                <button type="submit" style={{ background: '#0284c7', color: '#fff', border: 'none', padding: '0.75rem', borderRadius: '4px', fontWeight: 'bold', cursor: 'pointer' }}>Generate Secure Key</button>
              </form>

              {newlyCreatedKey && (
                <div style={{ marginTop: '1.5rem', padding: '1rem', background: '#0f172a', border: '1px dashed #0284c7', borderRadius: '6px' }}>
                  <span style={{ fontSize: '0.75rem', color: '#94a3b8', display: 'block' }}>COPY TOKEN NOW (WILL NOT BE SHOWN AGAIN)</span>
                  <code style={{ fontSize: '0.9rem', color: '#10b981', display: 'block', wordBreak: 'break-all', marginTop: '6px', fontWeight: 'bold' }}>{newlyCreatedKey}</code>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Tab 2: Providers Config */}
        {activeTab === 'providers' && (
          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr', gap: '2rem' }}>
            {/* List */}
            <div style={{ background: '#111827', border: '1px solid #1f2937', borderRadius: '8px', padding: '1.5rem' }}>
              <h3 style={{ margin: '0 0 1rem 0', color: '#f8fafc' }}>Configured Upstream Connections</h3>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                {providers.length === 0 ? (
                  <div style={{ textAlign: 'center', color: '#64748b', padding: '2rem' }}>No AI model provider nodes configured yet. Add one to start routing.</div>
                ) : (
                  providers.map((p) => (
                    <div key={p.id} style={{ background: '#0f172a', border: '1px solid #1f2937', padding: '1rem', borderRadius: '6px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <div>
                        <strong style={{ fontSize: '1rem', color: '#f8fafc' }}>{p.name}</strong>
                        <span style={{ display: 'block', fontSize: '0.75rem', color: '#94a3b8', marginTop: '4px' }}>Provider: <code style={{ color: '#a78bfa' }}>{p.provider}</code> | Endpoint: <code style={{ color: '#64748b' }}>{p.endpoint || 'Default Endpoint'}</code></span>
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                        <span style={{ fontSize: '0.8rem', background: '#1e293b', padding: '2px 8px', borderRadius: '4px' }}>Priority: {p.priority}</span>
                        <span style={{ padding: '2px 8px', borderRadius: '12px', background: p.is_active ? '#065f46' : '#991b1b', color: p.is_active ? '#34d399' : '#f87171', fontSize: '0.75rem', fontWeight: 'bold' }}>
                          {p.is_active ? 'ACTIVE' : 'DISABLED'}
                        </span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Create */}
            <div style={{ background: '#111827', border: '1px solid #1f2937', borderRadius: '8px', padding: '1.5rem' }}>
              <h3 style={{ margin: '0 0 1rem 0', color: '#f8fafc' }}>Add Provider Node</h3>
              <form onSubmit={handleAddProvider} style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
                <div>
                  <label style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginBottom: '6px' }}>Provider Platform</label>
                  <select value={providerId} onChange={(e) => setProviderId(e.target.value)} style={{ width: '100%', padding: '0.5rem', background: '#0b0f19', border: '1px solid #1f2937', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }}>
                    <option value="openai">OpenAI</option>
                    <option value="anthropic">Anthropic Claude</option>
                    <option value="google">Google Gemini</option>
                    <option value="perplexity">Perplexity AI</option>
                    <option value="openrouter">OpenRouter</option>
                  </select>
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginBottom: '6px' }}>Connection Label Name</label>
                  <input type="text" value={providerName} onChange={(e) => setProviderName(e.target.value)} placeholder="e.g. OpenAI Global Node" style={{ width: '100%', padding: '0.5rem', background: '#0b0f19', border: '1px solid #1f2937', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginBottom: '6px' }}>API Key / Auth Token</label>
                  <input type="password" value={providerKey} onChange={(e) => setProviderKey(e.target.value)} placeholder="sk-..." style={{ width: '100%', padding: '0.5rem', background: '#0b0f19', border: '1px solid #1f2937', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginBottom: '6px' }}>Custom Endpoint URL (Optional)</label>
                  <input type="text" value={providerEndpoint} onChange={(e) => setProviderEndpoint(e.target.value)} placeholder="https://..." style={{ width: '100%', padding: '0.5rem', background: '#0b0f19', border: '1px solid #1f2937', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginBottom: '6px' }}>Routing Priority Weight</label>
                  <input type="number" value={providerPriority} onChange={(e) => setProviderPriority(parseInt(e.target.value))} style={{ width: '100%', padding: '0.5rem', background: '#0b0f19', border: '1px solid #1f2937', borderRadius: '4px', color: '#fff', boxSizing: 'border-box' }} />
                </div>
                <button type="submit" style={{ background: '#10b981', color: '#fff', border: 'none', padding: '0.75rem', borderRadius: '4px', fontWeight: 'bold', cursor: 'pointer' }}>Register Node</button>
              </form>
            </div>
          </div>
        )}

        {/* Tab 3: Request Logs */}
        {activeTab === 'logs' && (
          <div style={{ background: '#111827', border: '1px solid #1f2937', borderRadius: '8px', padding: '1.5rem' }}>
            <h3 style={{ margin: '0 0 1rem 0', color: '#f8fafc' }}>Real-time Prompt Scrubbing & Audit Trail</h3>
            <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left' }}>
              <thead>
                <tr style={{ borderBottom: '1px solid #1f2937', color: '#94a3b8', fontSize: '0.8rem' }}>
                  <th style={{ padding: '0.75rem' }}>ID</th>
                  <th style={{ padding: '0.75rem' }}>Key Reference</th>
                  <th style={{ padding: '0.75rem' }}>Model</th>
                  <th style={{ padding: '0.75rem' }}>Prompt Tokens</th>
                  <th style={{ padding: '0.75rem' }}>Completion Output</th>
                  <th style={{ padding: '0.75rem' }}>Latency</th>
                  <th style={{ padding: '0.75rem' }}>App Origin</th>
                </tr>
              </thead>
              <tbody>
                {logs.length === 0 ? (
                  <tr><td colSpan={7} style={{ padding: '2rem', textAlign: 'center', color: '#64748b' }}>No requests audited yet. Initiate completion streaming to record traces.</td></tr>
                ) : (
                  logs.map((l) => (
                    <tr key={l.id} style={{ borderBottom: '1px solid #1f2937', fontSize: '0.875rem' }}>
                      <td style={{ padding: '0.75rem', color: '#64748b' }}>{l.id}</td>
                      <td style={{ padding: '0.75rem', color: '#38bdf8', fontFamily: 'monospace' }}>{l.key_id}</td>
                      <td style={{ padding: '0.75rem', fontWeight: 'bold' }}>{l.model_id}</td>
                      <td style={{ padding: '0.75rem', color: '#f43f5e' }}>{l.prompt_tokens} tokens (PII-Cleaned)</td>
                      <td style={{ padding: '0.75rem', color: '#8b5cf6' }}>{l.completion_tokens} tokens</td>
                      <td style={{ padding: '0.75rem', color: '#10b981', fontWeight: 'bold' }}>{l.latency_ms} ms</td>
                      <td style={{ padding: '0.75rem', color: '#a78bfa' }}>{l.source_app}</td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        )}

        {/* Tab 4: Model Catalog */}
        {activeTab === 'catalog' && (
          <div style={{ background: '#111827', border: '1px solid #1f2937', borderRadius: '8px', padding: '1.5rem' }}>
            <h3 style={{ margin: '0 0 1rem 0', color: '#f8fafc' }}>Active Gateway Routing Catalog</h3>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: '1.5rem', marginTop: '1rem' }}>
              {models.length === 0 ? (
                <div style={{ color: '#64748b', colSpan: 3, textAlign: 'center', padding: '2rem' }}>No models active. Register a provider to populate catalog.</div>
              ) : (
                models.map((m) => (
                  <div key={m.id} style={{ background: '#0f172a', border: '1px solid #1f2937', padding: '1.25rem', borderRadius: '6px' }}>
                    <strong style={{ fontSize: '1rem', color: '#38bdf8', display: 'block' }}>{m.id}</strong>
                    <span style={{ display: 'block', fontSize: '0.8rem', color: '#94a3b8', marginTop: '4px' }}>Canonical Name: {m.name || m.id}</span>
                    <span style={{ display: 'block', fontSize: '0.75rem', color: '#a78bfa', marginTop: '8px', background: '#1e293b', width: 'fit-content', padding: '2px 8px', borderRadius: '4px' }}>Provider: {m.provider}</span>
                  </div>
                ))
              )}
            </div>
          </div>
        )}

      </div>
    </div>
  );
}
