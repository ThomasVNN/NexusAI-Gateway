import { useState } from 'react';
import {
  Settings,
  Globe,
  DollarSign,
  Cog,
  Server,
  Save,
  Plus,
  Trash2,
} from 'lucide-react';

type SettingsSection = 'general' | 'model-pricing' | 'operations' | 'system';

interface ModelPricingSetting {
  id: string;
  model: string;
  provider: string;
  input_cost: number;
  output_cost: number;
  enabled: boolean;
}

interface OperationSetting {
  id: string;
  key: string;
  value: string | boolean | number;
  type: 'string' | 'boolean' | 'number';
  description: string;
}

const sections = [
  { id: 'general' as const, label: 'General', icon: Globe },
  { id: 'model-pricing' as const, label: 'Model Pricing', icon: DollarSign },
  { id: 'operations' as const, label: 'Operation Settings', icon: Cog },
  { id: 'system' as const, label: 'System', icon: Server },
];

export function SettingsPage() {
  const [activeSection, setActiveSection] = useState<SettingsSection>('general');
  const [saving, setSaving] = useState(false);

  const [generalSettings, setGeneralSettings] = useState({
    gatewayName: 'NexusAI Gateway',
    timezone: 'UTC',
    logRetention: '30',
    enableMetrics: true,
    enableAccessLogs: true,
  });

  const [modelPricing, setModelPricing] = useState<ModelPricingSetting[]>([
    { id: '1', model: 'GPT-4o', provider: 'openai', input_cost: 0.000005, output_cost: 0.000015, enabled: true },
    { id: '2', model: 'Claude 3.5 Sonnet', provider: 'anthropic', input_cost: 0.000003, output_cost: 0.000015, enabled: true },
    { id: '3', model: 'Gemini Pro', provider: 'google', input_cost: 0.00000125, output_cost: 0.000005, enabled: true },
  ]);

  const [operationSettings, setOperationSettings] = useState<OperationSetting[]>([
    { id: '1', key: 'max_concurrent_requests', value: 1000, type: 'number', description: 'Maximum concurrent API requests' },
    { id: '2', key: 'request_timeout_ms', value: 30000, type: 'number', description: 'Request timeout in milliseconds' },
    { id: '3', key: 'enable_caching', value: true, type: 'boolean', description: 'Enable response caching' },
    { id: '4', key: 'cache_ttl_seconds', value: 3600, type: 'number', description: 'Cache TTL in seconds' },
    { id: '5', key: 'retry_attempts', value: 3, type: 'number', description: 'Number of retry attempts on failure' },
    { id: '6', key: 'rate_limit_enabled', value: true, type: 'boolean', description: 'Enable rate limiting per key' },
  ]);

  const handleSave = async () => {
    setSaving(true);
    await new Promise((r) => setTimeout(r, 1000));
    setSaving(false);
  };

  const updateOperationSetting = (id: string, value: string | boolean | number) => {
    setOperationSettings((prev) =>
      prev.map((s) => (s.id === id ? { ...s, value } : s))
    );
  };

  const toggleModelEnabled = (id: string) => {
    setModelPricing((prev) =>
      prev.map((m) => (m.id === id ? { ...m, enabled: !m.enabled } : m))
    );
  };

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Settings</h1>
          <p className="text-sm text-text-tertiary mt-1">Configure your Gateway behavior and preferences</p>
        </div>
        <button
          onClick={handleSave}
          disabled={saving}
          className="flex items-center gap-2 px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors disabled:opacity-50"
        >
          <Save className="w-4 h-4" />
          {saving ? 'Saving...' : 'Save Changes'}
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Section Navigation */}
        <div className="lg:col-span-1">
          <nav className="bg-bg-tertiary border border-border-subtle rounded-xl p-2 space-y-1">
            {sections.map((section) => (
              <button
                key={section.id}
                onClick={() => setActiveSection(section.id)}
                className={`w-full flex items-center gap-3 px-4 py-3 rounded-lg text-sm font-medium transition-colors ${
                  activeSection === section.id
                    ? 'bg-accent-primary/10 text-accent-primary'
                    : 'text-text-secondary hover:text-text-primary hover:bg-bg-elevated'
                }`}
              >
                <section.icon className="w-5 h-5" />
                {section.label}
              </button>
            ))}
          </nav>
        </div>

        {/* Section Content */}
        <div className="lg:col-span-3">
          {/* General Settings */}
          {activeSection === 'general' && (
            <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-6 space-y-6">
              <h2 className="text-lg font-semibold text-text-primary flex items-center gap-2">
                <Globe className="w-5 h-5" />
                General Settings
              </h2>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1">Gateway Name</label>
                  <input
                    type="text"
                    value={generalSettings.gatewayName}
                    onChange={(e) => setGeneralSettings((prev) => ({ ...prev, gatewayName: e.target.value }))}
                    className="w-full px-3 py-2 bg-bg-secondary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1">Timezone</label>
                  <select
                    value={generalSettings.timezone}
                    onChange={(e) => setGeneralSettings((prev) => ({ ...prev, timezone: e.target.value }))}
                    className="w-full px-3 py-2 bg-bg-secondary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                  >
                    <option value="UTC">UTC</option>
                    <option value="America/New_York">Eastern Time</option>
                    <option value="America/Los_Angeles">Pacific Time</option>
                    <option value="Europe/London">London</option>
                    <option value="Asia/Tokyo">Tokyo</option>
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1">Log Retention (days)</label>
                  <input
                    type="number"
                    value={generalSettings.logRetention}
                    onChange={(e) => setGeneralSettings((prev) => ({ ...prev, logRetention: e.target.value }))}
                    className="w-full px-3 py-2 bg-bg-secondary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                  />
                </div>

                <div className="space-y-3 pt-4 border-t border-border-subtle">
                  <label className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-text-primary">Enable Metrics</p>
                      <p className="text-xs text-text-muted">Collect and display usage metrics</p>
                    </div>
                    <input
                      type="checkbox"
                      checked={generalSettings.enableMetrics}
                      onChange={(e) => setGeneralSettings((prev) => ({ ...prev, enableMetrics: e.target.checked }))}
                      className="w-5 h-5 rounded border-border-subtle bg-bg-secondary text-accent-primary focus:ring-accent-primary"
                    />
                  </label>

                  <label className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium text-text-primary">Enable Access Logs</p>
                      <p className="text-xs text-text-muted">Log all API access requests</p>
                    </div>
                    <input
                      type="checkbox"
                      checked={generalSettings.enableAccessLogs}
                      onChange={(e) => setGeneralSettings((prev) => ({ ...prev, enableAccessLogs: e.target.checked }))}
                      className="w-5 h-5 rounded border-border-subtle bg-bg-secondary text-accent-primary focus:ring-accent-primary"
                    />
                  </label>
                </div>
              </div>
            </div>
          )}

          {/* Model Pricing */}
          {activeSection === 'model-pricing' && (
            <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-6 space-y-6">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold text-text-primary flex items-center gap-2">
                  <DollarSign className="w-5 h-5" />
                  Model Pricing
                </h2>
                <button className="flex items-center gap-2 px-3 py-1.5 text-sm bg-accent-primary/10 text-accent-primary rounded-lg hover:bg-accent-primary/20 transition-colors">
                  <Plus className="w-4 h-4" />
                  Add Model
                </button>
              </div>

              <div className="space-y-4">
                {modelPricing.map((model) => (
                  <div
                    key={model.id}
                    className={`p-4 border rounded-xl transition-colors ${
                      model.enabled ? 'border-border-subtle' : 'border-border-subtle opacity-60'
                    }`}
                  >
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <input
                          type="checkbox"
                          checked={model.enabled}
                          onChange={() => toggleModelEnabled(model.id)}
                          className="w-4 h-4 rounded border-border-subtle bg-bg-secondary text-accent-primary focus:ring-accent-primary"
                        />
                        <div>
                          <p className="font-medium text-text-primary">{model.model}</p>
                          <p className="text-xs text-text-muted">{model.provider}</p>
                        </div>
                      </div>
                      <button className="text-text-muted hover:text-error transition-colors">
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <label className="block text-xs text-text-muted mb-1">Input Cost (per 1K tokens)</label>
                        <input
                          type="number"
                          step="0.000001"
                          value={model.input_cost}
                          disabled={!model.enabled}
                          className="w-full px-3 py-2 bg-bg-secondary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none disabled:opacity-50"
                        />
                      </div>
                      <div>
                        <label className="block text-xs text-text-muted mb-1">Output Cost (per 1K tokens)</label>
                        <input
                          type="number"
                          step="0.000001"
                          value={model.output_cost}
                          disabled={!model.enabled}
                          className="w-full px-3 py-2 bg-bg-secondary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none disabled:opacity-50"
                        />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Operation Settings */}
          {activeSection === 'operations' && (
            <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-6 space-y-6">
              <h2 className="text-lg font-semibold text-text-primary flex items-center gap-2">
                <Cog className="w-5 h-5" />
                Operation Settings
              </h2>

              <div className="space-y-4">
                {operationSettings.map((setting) => (
                  <div key={setting.id} className="flex items-center justify-between py-3 border-b border-border-subtle last:border-0">
                    <div>
                      <p className="text-sm font-medium text-text-primary">{setting.key.replace(/_/g, ' ')}</p>
                      <p className="text-xs text-text-muted">{setting.description}</p>
                    </div>
                    <div className="w-48">
                      {setting.type === 'boolean' ? (
                        <input
                          type="checkbox"
                          checked={setting.value as boolean}
                          onChange={(e) => updateOperationSetting(setting.id, e.target.checked)}
                          className="w-5 h-5 rounded border-border-subtle bg-bg-secondary text-accent-primary focus:ring-accent-primary"
                        />
                      ) : (
                        <input
                          type="number"
                          value={setting.value as number}
                          onChange={(e) => updateOperationSetting(setting.id, Number(e.target.value))}
                          className="w-full px-3 py-2 bg-bg-secondary border border-border-subtle rounded-lg text-text-primary text-right focus:border-border-focus focus:outline-none"
                        />
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* System Settings */}
          {activeSection === 'system' && (
            <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-6 space-y-6">
              <h2 className="text-lg font-semibold text-text-primary flex items-center gap-2">
                <Server className="w-5 h-5" />
                System Information
              </h2>

              <div className="grid grid-cols-2 gap-4">
                {[
                  { label: 'Version', value: '2.0.0' },
                  { label: 'Go Version', value: '1.22.0' },
                  { label: 'Database', value: 'PostgreSQL 15' },
                  { label: 'Redis', value: '7.2.0' },
                  { label: 'Uptime', value: '14 days, 6 hours' },
                  { label: 'Environment', value: 'Production' },
                ].map((item, i) => (
                  <div key={i} className="p-4 bg-bg-secondary rounded-xl">
                    <p className="text-xs text-text-muted">{item.label}</p>
                    <p className="text-sm font-medium text-text-primary mt-1">{item.value}</p>
                  </div>
                ))}
              </div>

              <div className="pt-4 border-t border-border-subtle">
                <h3 className="text-sm font-medium text-text-primary mb-3">Danger Zone</h3>
                <div className="space-y-2">
                  <button className="w-full px-4 py-3 text-left text-sm text-error hover:bg-error/10 rounded-lg transition-colors">
                    Clear All Logs
                  </button>
                  <button className="w-full px-4 py-3 text-left text-sm text-error hover:bg-error/10 rounded-lg transition-colors">
                    Reset All Settings
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
