import React, { useState, useEffect } from 'react';
import { KPIGrid, AIInsightsPanel, ActivityTimeline, WorkspaceOverview, type KPIData, type Insight, type ActivityItem, type WorkspaceOverviewItem } from './components';

// ============================================================================
// Type Definitions
// ============================================================================

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

// ============================================================================
// Icons
// ============================================================================

function NexusAILogo() {
  return (
    <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-accent-primary to-accent-secondary flex items-center justify-center">
      <span className="text-white font-bold text-sm">NX</span>
    </div>
  );
}

function RequestIcon() {
  return (
    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
    </svg>
  );
}

function LatencyIcon() {
  return (
    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
    </svg>
  );
}

function TokenIcon() {
  return (
    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M7 8h10M7 12h4m1 8l-4-4H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-3l-4 4z" />
    </svg>
  );
}

function OutputIcon() {
  return (
    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
    </svg>
  );
}

// ============================================================================
// Main App Component
// ============================================================================

export default function App() {
  const [activeTab, setActiveTab] = useState<'dashboard' | 'keys' | 'providers' | 'logs' | 'catalog'>('dashboard');

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

  // ============================================================================
  // Data Loading
  // ============================================================================

  const loadData = async () => {
    try {
      const [keysRes, provRes, detailLogsRes, logsRes, modelsRes, versionRes] = await Promise.all([
        fetch('/api/admin/keys'),
        fetch('/api/providers'),
        fetch('/api/admin/logs'),
        fetch('/api/admin/usage'),
        fetch('/api/models'),
        fetch('/api/system/version'),
      ]);

      if (keysRes.ok) setKeys(await keysRes.json());
      if (provRes.ok) {
        const provData = await provRes.json();
        setProviders(provData.connections || []);
      }
      if (detailLogsRes.ok) setLogs(await detailLogsRes.json());
      if (logsRes.ok) {
        const usageData = await logsRes.json();
        setUsage(usageData || { total_calls: 0, total_prompt_tokens: 0, total_completion_tokens: 0, average_latency_ms: 0 });
      }
      if (modelsRes.ok) {
        const modelsData = await modelsRes.json();
        setModels(modelsData.models || []);
      }
      if (versionRes.ok) setSysVersion(await versionRes.json());
    } catch (e) {
      console.error('Failed to reload admin telemetry:', e);
    }
  };

  useEffect(() => {
    loadData();
    const interval = setInterval(loadData, 5000);
    return () => clearInterval(interval);
  }, []);

  // ============================================================================
  // Form Handlers
  // ============================================================================

  const handleCreateKey = async (e: React.FormEvent) => {
    e.preventDefault();
    setNewlyCreatedKey(null);
    setMessage('');
    try {
      const res = await fetch('/api/admin/keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: keyName, source_app: sourceApp, daily_quota: dailyQuota, hourly_quota: hourlyQuota }),
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
    } catch {
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
        body: JSON.stringify({ id: providerId, provider: providerId, name: providerName || `${providerId} Connection`, api_key: providerKey, endpoint: providerEndpoint, is_active: true, priority: providerPriority }),
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
    } catch {
      setMessage('Network error.');
    }
  };

  // ============================================================================
  // Transform Data for Components
  // ============================================================================

  const kpiData: KPIData[] = [
    {
      id: 'total-requests',
      label: 'Total Requests',
      value: usage.total_calls,
      trend: { direction: 'up', percentage: 12, label: 'vs last hour' },
      icon: <RequestIcon />,
      accentColor: 'purple',
    },
    {
      id: 'avg-latency',
      label: 'Avg Latency',
      value: `${usage.average_latency_ms.toFixed(0)}ms`,
      trend: { direction: 'down', percentage: 8, label: 'vs last hour' },
      icon: <LatencyIcon />,
      accentColor: 'green',
    },
    {
      id: 'prompt-tokens',
      label: 'Prompt Tokens',
      value: usage.total_prompt_tokens,
      trend: { direction: 'up', percentage: 23, label: 'vs last hour' },
      icon: <TokenIcon />,
      accentColor: 'blue',
    },
    {
      id: 'completion-output',
      label: 'Completion Output',
      value: usage.total_completion_tokens,
      trend: { direction: 'neutral', percentage: 0, label: 'vs last hour' },
      icon: <OutputIcon />,
      accentColor: 'yellow',
    },
  ];

  const insightsData: Insight[] = [
    {
      id: 'insight-1',
      type: 'recommendation',
      title: 'Consider enabling caching',
      description: '22% of requests are duplicates. Implementing response caching could reduce costs by up to 15%.',
      action: { label: 'Configure caching', onClick: () => setActiveTab('dashboard') },
      timestamp: '2 min ago',
    },
    {
      id: 'insight-2',
      type: 'warning',
      title: 'Latency spike detected',
      description: 'Average latency increased by 12% in the last 15 minutes on GPT-4o requests.',
      action: { label: 'View details', onClick: () => setActiveTab('logs') },
      timestamp: '15 min ago',
    },
    {
      id: 'insight-3',
      type: 'success',
      title: 'Cost optimization working',
      description: 'Switching to Claude 3.5 Sonnet for complex reasoning tasks saved $234 this week.',
      timestamp: '1 hour ago',
    },
  ];

  const activityData: ActivityItem[] = logs.slice(0, 5).map((log) => ({
    id: String(log.id),
    type: 'api_call',
    title: `${log.source_app} → ${log.model_id}`,
    description: `${log.prompt_tokens} prompt tokens, ${log.completion_tokens} completion tokens`,
    timestamp: log.created_at,
    metadata: { latency: `${log.latency_ms}ms` },
  }));

  const workspaceData: WorkspaceOverviewItem[] = providers.map((p) => ({
    id: p.id,
    name: p.name,
    status: p.is_active ? 'healthy' : 'down',
    type: 'provider',
    metrics: [
      { label: 'Priority', value: p.priority, status: 'good' },
      { label: 'Provider', value: p.provider, status: 'good' },
    ],
    lastActivity: 'Active now',
  }));

  // ============================================================================
  // Render
  // ============================================================================

  return (
    <div className="min-h-screen bg-bg-primary">
      {/* Header */}
      <header className="sticky top-0 z-50 bg-bg-secondary/80 backdrop-blur-md border-b border-border-subtle">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            {/* Logo */}
            <div className="flex items-center gap-3">
              <NexusAILogo />
              <div>
                <h1 className="text-lg font-bold gradient-text">NexusAI-Gateway</h1>
                <p className="text-xs text-text-tertiary">v{sysVersion.version} • {sysVersion.engine}</p>
              </div>
            </div>

            {/* Navigation */}
            <nav className="flex items-center gap-1">
              {[
                { id: 'dashboard', label: 'Dashboard' },
                { id: 'keys', label: 'API Keys' },
                { id: 'providers', label: 'Providers' },
                { id: 'logs', label: 'Logs' },
                { id: 'catalog', label: 'Catalog' },
              ].map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActiveTab(tab.id as typeof activeTab)}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-all duration-150 ${
                    activeTab === tab.id
                      ? 'bg-accent-primary/10 text-accent-primary'
                      : 'text-text-secondary hover:text-text-primary hover:bg-bg-elevated'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </nav>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Dashboard View */}
        {activeTab === 'dashboard' && (
          <div className="space-y-8 animate-fade-in">
            {/* KPI Section */}
            <section>
              <h2 className="text-xl font-semibold text-text-primary mb-4">Overview</h2>
              <KPIGrid items={kpiData} columns={4} />
            </section>

            {/* Two Column Layout */}
            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              {/* Left: AI Insights */}
              <div className="lg:col-span-2">
                <AIInsightsPanel insights={insightsData} />
              </div>

              {/* Right: Activity */}
              <div>
                <ActivityTimeline activities={activityData} maxItems={5} />
              </div>
            </div>

            {/* Workspace Overview */}
            <section>
              <WorkspaceOverview items={workspaceData} />
            </section>
          </div>
        )}

        {/* API Keys View */}
        {activeTab === 'keys' && (
          <div className="space-y-6 animate-fade-in">
            {message && (
              <div className="flex items-center justify-between px-4 py-3 bg-accent-primary/10 border border-accent-primary/30 rounded-lg">
                <span className="text-sm text-text-primary">{message}</span>
                <button onClick={() => setMessage('')} className="text-text-tertiary hover:text-text-primary">
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            )}

            <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
              {/* Table */}
              <div className="lg:col-span-2 panel">
                <div className="panel-header">
                  <h3 className="text-lg font-semibold text-text-primary">Authorized Client Keys</h3>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border-subtle">
                        <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">ID</th>
                        <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Name</th>
                        <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Source</th>
                        <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Daily Quota</th>
                        <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Status</th>
                      </tr>
                    </thead>
                    <tbody>
                      {keys.length === 0 ? (
                        <tr>
                          <td colSpan={5} className="px-4 py-8 text-center text-text-tertiary">
                            No keys active. Generate a token to get started.
                          </td>
                        </tr>
                      ) : (
                        keys.map((k) => (
                          <tr key={k.id} className="border-b border-border-subtle hover:bg-bg-elevated/50 transition-colors">
                            <td className="px-4 py-3 font-mono text-accent-primary">{k.id.slice(0, 12)}...</td>
                            <td className="px-4 py-3 font-medium text-text-primary">{k.name}</td>
                            <td className="px-4 py-3 text-text-secondary">{k.source_app}</td>
                            <td className="px-4 py-3 text-text-secondary">{k.daily_quota.toLocaleString()}</td>
                            <td className="px-4 py-3">
                              <span className={`badge ${k.active ? 'badge-success' : 'badge-error'}`}>
                                {k.active ? 'Active' : 'Revoked'}
                              </span>
                            </td>
                          </tr>
                        ))
                      )}
                    </tbody>
                  </table>
                </div>
              </div>

              {/* Create Form */}
              <div className="panel">
                <div className="panel-header">
                  <h3 className="text-lg font-semibold text-text-primary">Create Token</h3>
                </div>
                <form onSubmit={handleCreateKey} className="p-5 space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-text-secondary mb-1.5">Key Description</label>
                    <input
                      type="text"
                      value={keyName}
                      onChange={(e) => setKeyName(e.target.value)}
                      required
                      placeholder="e.g. Cursor-Copilot-A"
                      className="input"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-text-secondary mb-1.5">Source App</label>
                    <select value={sourceApp} onChange={(e) => setSourceApp(e.target.value)} className="input">
                      <option value="openwebui">OpenWebUI</option>
                      <option value="openclaude">OpenClaude</option>
                      <option value="codex">Codex</option>
                      <option value="antigravity">Antigravity</option>
                      <option value="direct-api">Direct API</option>
                    </select>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-text-secondary mb-1.5">Daily Quota</label>
                      <input
                        type="number"
                        value={dailyQuota}
                        onChange={(e) => setDailyQuota(parseInt(e.target.value))}
                        className="input"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-text-secondary mb-1.5">Hourly Quota</label>
                      <input
                        type="number"
                        value={hourlyQuota}
                        onChange={(e) => setHourlyQuota(parseInt(e.target.value))}
                        className="input"
                      />
                    </div>
                  </div>
                  <button type="submit" className="btn btn-primary w-full">
                    Generate Secure Key
                  </button>
                </form>

                {newlyCreatedKey && (
                  <div className="mx-5 mb-5 p-4 bg-warning/10 border border-warning/30 rounded-lg">
                    <p className="text-xs font-medium text-warning mb-2">COPY TOKEN NOW (WILL NOT BE SHOWN AGAIN)</p>
                    <code className="text-sm text-success break-all">{newlyCreatedKey}</code>
                  </div>
                )}
              </div>
            </div>
          </div>
        )}

        {/* Providers View */}
        {activeTab === 'providers' && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 animate-fade-in">
            {/* List */}
            <div className="lg:col-span-2 panel">
              <div className="panel-header">
                <h3 className="text-lg font-semibold text-text-primary">Configured Providers</h3>
              </div>
              <div className="p-5 space-y-4">
                {providers.length === 0 ? (
                  <div className="text-center py-8 text-text-tertiary">
                    No providers configured. Add one to start routing.
                  </div>
                ) : (
                  providers.map((p) => (
                    <div key={p.id} className="flex items-center justify-between p-4 bg-bg-elevated rounded-lg">
                      <div>
                        <h4 className="font-medium text-text-primary">{p.name}</h4>
                        <p className="text-sm text-text-tertiary">
                          <span className="text-accent-secondary">{p.provider}</span> • {p.endpoint || 'Default Endpoint'}
                        </p>
                      </div>
                      <div className="flex items-center gap-3">
                        <span className="badge badge-info">Priority {p.priority}</span>
                        <span className={`badge ${p.is_active ? 'badge-success' : 'badge-error'}`}>
                          {p.is_active ? 'Active' : 'Disabled'}
                        </span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Add Form */}
            <div className="panel">
              <div className="panel-header">
                <h3 className="text-lg font-semibold text-text-primary">Add Provider</h3>
              </div>
              <form onSubmit={handleAddProvider} className="p-5 space-y-4">
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1.5">Platform</label>
                  <select value={providerId} onChange={(e) => setProviderId(e.target.value)} className="input">
                    <option value="openai">OpenAI</option>
                    <option value="anthropic">Anthropic Claude</option>
                    <option value="google">Google Gemini</option>
                    <option value="perplexity">Perplexity AI</option>
                    <option value="openrouter">OpenRouter</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1.5">Label</label>
                  <input
                    type="text"
                    value={providerName}
                    onChange={(e) => setProviderName(e.target.value)}
                    placeholder="e.g. OpenAI Global"
                    className="input"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1.5">API Key</label>
                  <input
                    type="password"
                    value={providerKey}
                    onChange={(e) => setProviderKey(e.target.value)}
                    placeholder="sk-..."
                    className="input"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1.5">Endpoint (Optional)</label>
                  <input
                    type="text"
                    value={providerEndpoint}
                    onChange={(e) => setProviderEndpoint(e.target.value)}
                    placeholder="https://..."
                    className="input"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1.5">Priority</label>
                  <input
                    type="number"
                    value={providerPriority}
                    onChange={(e) => setProviderPriority(parseInt(e.target.value))}
                    className="input"
                  />
                </div>
                <button type="submit" className="btn btn-primary w-full">
                  Register Provider
                </button>
              </form>
            </div>
          </div>
        )}

        {/* Logs View */}
        {activeTab === 'logs' && (
          <div className="panel animate-fade-in">
            <div className="panel-header">
              <h3 className="text-lg font-semibold text-text-primary">Request Logs</h3>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border-subtle">
                    <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">ID</th>
                    <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Key</th>
                    <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Model</th>
                    <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Prompt Tokens</th>
                    <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Completion</th>
                    <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Latency</th>
                    <th className="text-left px-4 py-3 text-xs font-medium text-text-tertiary">Source</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="px-4 py-8 text-center text-text-tertiary">
                        No logs yet. Make API requests to see them here.
                      </td>
                    </tr>
                  ) : (
                    logs.map((l) => (
                      <tr key={l.id} className="border-b border-border-subtle hover:bg-bg-elevated/50 transition-colors">
                        <td className="px-4 py-3 text-text-muted">{l.id}</td>
                        <td className="px-4 py-3 font-mono text-accent-primary">{l.key_id.slice(0, 12)}...</td>
                        <td className="px-4 py-3 font-medium text-text-primary">{l.model_id}</td>
                        <td className="px-4 py-3 text-error">{l.prompt_tokens}</td>
                        <td className="px-4 py-3 text-accent-primary">{l.completion_tokens}</td>
                        <td className="px-4 py-3 text-success font-medium">{l.latency_ms}ms</td>
                        <td className="px-4 py-3 text-text-secondary">{l.source_app}</td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {/* Catalog View */}
        {activeTab === 'catalog' && (
          <div className="panel animate-fade-in">
            <div className="panel-header">
              <h3 className="text-lg font-semibold text-text-primary">Model Catalog</h3>
            </div>
            <div className="p-5">
              {models.length === 0 ? (
                <div className="text-center py-8 text-text-tertiary">
                  No models available. Register a provider to populate the catalog.
                </div>
              ) : (
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  {models.map((m) => (
                    <div key={m.id} className="p-4 bg-bg-elevated rounded-lg border border-border-subtle hover:border-accent-primary/30 transition-colors">
                      <h4 className="font-mono text-accent-primary mb-1">{m.id}</h4>
                      <p className="text-sm text-text-secondary mb-2">{m.name || m.id}</p>
                      <span className="badge badge-info">{m.provider}</span>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}
      </main>
    </div>
  );
}
