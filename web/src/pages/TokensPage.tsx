import { useState, useEffect } from 'react';
import {
  Plus,
  Key,
  Copy,
  Trash2,
  Eye,
  EyeOff,
  MoreHorizontal,
  Edit,
  ToggleLeft,
  ToggleRight,
  Search,
} from 'lucide-react';
import { EnhancedTable, type Column } from '../tables/EnhancedTable';

interface Token {
  id: string;
  name: string;
  key: string;
  group?: string;
  daily_quota: number;
  hourly_quota: number;
  used_today: number;
  used_this_hour: number;
  is_active: boolean;
  source_app: string;
  created_at: string;
  last_used: string;
}

const sourceApps = ['OpenWebUI', 'OpenClaude', 'Codex', 'Antigravity', 'Direct API'];
const tokenGroups = ['Production', 'Development', 'Testing', 'Staging'];

export function TokensPage() {
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showKeyModal, setShowKeyModal] = useState(false);
  const [newlyCreatedKey, setNewlyCreatedKey] = useState<string | null>(null);
  const [visibleKeys, setVisibleKeys] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState('');
  const [groupFilter, setGroupFilter] = useState<string>('all');

  useEffect(() => {
    fetchTokens();
  }, []);

  const fetchTokens = async () => {
    try {
      const res = await fetch('/api/admin/keys');
      if (res.ok) {
        const data = await res.json();
        setTokens(data);
      }
    } catch {
      setTokens([
        {
          id: 'key-1',
          name: 'Production Primary',
          key: 'nxg_prod_abc123def456ghi789',
          group: 'Production',
          daily_quota: 100000,
          hourly_quota: 5000,
          used_today: 24500,
          used_this_hour: 890,
          is_active: true,
          source_app: 'OpenWebUI',
          created_at: '2024-01-15T10:00:00Z',
          last_used: '2024-05-31T10:30:00Z',
        },
        {
          id: 'key-2',
          name: 'Dev Environment',
          key: 'nxg_dev_xyz789uvw456rst123',
          group: 'Development',
          daily_quota: 10000,
          hourly_quota: 500,
          used_today: 3200,
          used_this_hour: 245,
          is_active: true,
          source_app: 'Codex',
          created_at: '2024-02-01T14:30:00Z',
          last_used: '2024-05-31T09:15:00Z',
        },
        {
          id: 'key-3',
          name: 'Staging API',
          key: 'nxg_stg_mno321pqr654stu987',
          group: 'Staging',
          daily_quota: 50000,
          hourly_quota: 2500,
          used_today: 12500,
          used_this_hour: 420,
          is_active: true,
          source_app: 'Antigravity',
          created_at: '2024-03-01T09:00:00Z',
          last_used: '2024-05-30T22:45:00Z',
        },
        {
          id: 'key-4',
          name: 'Legacy Key',
          key: 'nxg_old_lmn111aaa222bbb333',
          group: 'Production',
          daily_quota: 5000,
          hourly_quota: 200,
          used_today: 890,
          used_this_hour: 45,
          is_active: false,
          source_app: 'Direct API',
          created_at: '2023-06-15T12:00:00Z',
          last_used: '2024-04-20T16:30:00Z',
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const toggleToken = async (token: Token) => {
    const updated = { ...token, is_active: !token.is_active };
    try {
      await fetch(`/api/admin/keys/${token.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: updated.is_active }),
      });
      setTokens((prev) => prev.map((t) => (t.id === token.id ? updated : t)));
    } catch (error) {
      console.error('Failed to toggle token:', error);
    }
  };

  const deleteToken = async (token: Token) => {
    if (!confirm(`Delete API key "${token.name}"? This action cannot be undone.`)) return;
    try {
      await fetch(`/api/admin/keys/${token.id}`, { method: 'DELETE' });
      setTokens((prev) => prev.filter((t) => t.id !== token.id));
    } catch (error) {
      console.error('Failed to delete token:', error);
    }
  };

  const copyKey = (key: string) => {
    navigator.clipboard.writeText(key);
  };

  const toggleKeyVisibility = (id: string) => {
    setVisibleKeys((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  const filteredTokens = tokens.filter((token) => {
    const matchesSearch = token.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      token.key.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesGroup = groupFilter === 'all' || token.group === groupFilter;
    return matchesSearch && matchesGroup;
  });

  const columns: Column<Token>[] = [
    {
      key: 'name',
      header: 'API Key',
      sortable: true,
      render: (row) => (
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-accent-primary/10 flex items-center justify-center">
            <Key className="w-5 h-5 text-accent-primary" />
          </div>
          <div>
            <p className="font-medium text-text-primary">{row.name}</p>
            <p className="text-xs font-mono text-text-muted">
              {visibleKeys.has(row.id) ? row.key : `${row.key.slice(0, 20)}...`}
            </p>
          </div>
        </div>
      ),
    },
    {
      key: 'group',
      header: 'Group',
      width: '120px',
      render: (row) => (
        <span className="px-2 py-1 rounded-full text-xs font-medium bg-bg-elevated text-text-secondary">
          {row.group || 'Ungrouped'}
        </span>
      ),
    },
    {
      key: 'quota',
      header: 'Usage',
      width: '180px',
      render: (row) => {
        const dailyPercent = (row.used_today / row.daily_quota) * 100;
        const hourlyPercent = (row.used_this_hour / row.hourly_quota) * 100;
        return (
          <div className="space-y-2">
            <div>
              <div className="flex justify-between text-xs mb-1">
                <span className="text-text-muted">Daily</span>
                <span className="text-text-secondary">{row.used_today.toLocaleString()} / {row.daily_quota.toLocaleString()}</span>
              </div>
              <div className="w-32 h-1.5 bg-bg-elevated rounded-full overflow-hidden">
                <div
                  className={`h-full rounded-full ${dailyPercent > 90 ? 'bg-error' : dailyPercent > 70 ? 'bg-warning' : 'bg-success'}`}
                  style={{ width: `${Math.min(dailyPercent, 100)}%` }}
                />
              </div>
            </div>
          </div>
        );
      },
    },
    {
      key: 'is_active',
      header: 'Status',
      width: '100px',
      render: (row) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            toggleToken(row);
          }}
          className="flex items-center gap-2"
        >
          {row.is_active ? (
            <ToggleRight className="w-5 h-5 text-success" />
          ) : (
            <ToggleLeft className="w-5 h-5 text-text-muted" />
          )}
          <span className={`text-sm ${row.is_active ? 'text-success' : 'text-text-muted'}`}>
            {row.is_active ? 'Active' : 'Inactive'}
          </span>
        </button>
      ),
    },
    {
      key: 'last_used',
      header: 'Last Used',
      width: '140px',
      render: (row) => (
        <span className="text-sm text-text-secondary">
          {new Date(row.last_used).toLocaleDateString()}
        </span>
      ),
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">API Keys</h1>
          <p className="text-sm text-text-tertiary mt-1">Manage your API keys and access tokens</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="flex items-center gap-2 px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors"
        >
          <Plus className="w-4 h-4" />
          Create API Key
        </button>
      </div>

      {/* Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search keys..."
            className="w-full pl-10 pr-4 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:border-border-focus focus:outline-none"
          />
        </div>
        <select
          value={groupFilter}
          onChange={(e) => setGroupFilter(e.target.value)}
          className="px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary text-sm focus:border-border-focus focus:outline-none"
        >
          <option value="all">All Groups</option>
          {tokenGroups.map((group) => (
            <option key={group} value={group}>{group}</option>
          ))}
        </select>
      </div>

      {/* Tokens Table */}
      <EnhancedTable
        columns={columns}
        data={filteredTokens}
        loading={loading}
        onDelete={deleteToken}
        emptyMessage="No API keys found. Create your first key to get started."
      />

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-bg-secondary border border-border-subtle rounded-xl p-6 w-full max-w-lg">
            <h2 className="text-lg font-semibold text-text-primary mb-4">Create API Key</h2>
            <form className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Key Name</label>
                <input
                  type="text"
                  placeholder="e.g. Production Primary"
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:border-border-focus focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Group</label>
                <select className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none">
                  <option value="">No Group</option>
                  {tokenGroups.map((group) => (
                    <option key={group} value={group}>{group}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Source App</label>
                <select className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none">
                  {sourceApps.map((app) => (
                    <option key={app} value={app}>{app}</option>
                  ))}
                </select>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1">Daily Quota</label>
                  <input
                    type="number"
                    defaultValue={10000}
                    className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1">Hourly Quota</label>
                  <input
                    type="number"
                    defaultValue={500}
                    className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 text-text-secondary hover:text-text-primary transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  onClick={(e) => {
                    e.preventDefault();
                    setShowCreateModal(false);
                    setNewlyCreatedKey('nxg_new_' + Math.random().toString(36).substring(2, 20));
                    setShowKeyModal(true);
                  }}
                  className="px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors"
                >
                  Create Key
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Key Display Modal */}
      {showKeyModal && newlyCreatedKey && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-bg-secondary border border-border-subtle rounded-xl p-6 w-full max-w-lg">
            <div className="text-center mb-6">
              <div className="w-12 h-12 mx-auto mb-4 rounded-full bg-warning/10 flex items-center justify-center">
                <Key className="w-6 h-6 text-warning" />
              </div>
              <h2 className="text-lg font-semibold text-text-primary mb-2">API Key Created</h2>
              <p className="text-sm text-text-tertiary">
                Copy this key now. You will not be able to see it again.
              </p>
            </div>
            <div className="bg-bg-tertiary rounded-lg p-4 mb-4">
              <code className="text-sm font-mono text-text-primary break-all">{newlyCreatedKey}</code>
            </div>
            <div className="flex gap-3">
              <button
                onClick={() => copyKey(newlyCreatedKey)}
                className="flex-1 flex items-center justify-center gap-2 px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors"
              >
                <Copy className="w-4 h-4" />
                Copy Key
              </button>
              <button
                onClick={() => {
                  setShowKeyModal(false);
                  setNewlyCreatedKey(null);
                }}
                className="px-4 py-2 text-text-secondary hover:text-text-primary transition-colors"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
