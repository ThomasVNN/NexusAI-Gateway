import { useState, useEffect } from 'react';
import {
  Plus,
  Network,
  MoreHorizontal,
  Edit,
  Trash2,
  ToggleLeft,
  ToggleRight,
  Gauge,
} from 'lucide-react';
import { EnhancedTable, type Column } from '../tables/EnhancedTable';

interface Channel {
  id: string;
  name: string;
  provider: string;
  endpoint: string;
  priority: number;
  ratio: number;
  is_active: boolean;
  model_mapping: Record<string, string>;
  created_at: string;
}

export function ChannelsPage() {
  const [channels, setChannels] = useState<Channel[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingChannel, setEditingChannel] = useState<Channel | null>(null);

  useEffect(() => {
    fetchChannels();
  }, []);

  const fetchChannels = async () => {
    try {
      const res = await fetch('/api/providers');
      if (res.ok) {
        const data = await res.json();
        setChannels(data.connections || []);
      }
    } catch {
      setChannels([
        {
          id: 'ch-1',
          name: 'OpenAI Global',
          provider: 'openai',
          endpoint: 'https://api.openai.com/v1',
          priority: 1,
          ratio: 60,
          is_active: true,
          model_mapping: { 'gpt-4': 'gpt-4', 'gpt-4o': 'gpt-4o' },
          created_at: '2024-01-15T10:00:00Z',
        },
        {
          id: 'ch-2',
          name: 'Anthropic Primary',
          provider: 'anthropic',
          endpoint: 'https://api.anthropic.com/v1',
          priority: 2,
          ratio: 30,
          is_active: true,
          model_mapping: { 'claude-3': 'claude-3-opus', 'claude-3.5': 'claude-3-5-sonnet' },
          created_at: '2024-01-16T14:30:00Z',
        },
        {
          id: 'ch-3',
          name: 'Google Gemini',
          provider: 'google',
          endpoint: 'https://generativelanguage.googleapis.com/v1',
          priority: 3,
          ratio: 10,
          is_active: false,
          model_mapping: { 'gemini-pro': 'gemini-pro' },
          created_at: '2024-02-01T09:00:00Z',
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const toggleChannel = async (channel: Channel) => {
    const updated = { ...channel, is_active: !channel.is_active };
    try {
      await fetch(`/api/providers/${channel.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: updated.is_active }),
      });
      setChannels((prev) => prev.map((c) => (c.id === channel.id ? updated : c)));
    } catch (error) {
      console.error('Failed to toggle channel:', error);
    }
  };

  const deleteChannel = async (channel: Channel) => {
    if (!confirm(`Delete channel "${channel.name}"?`)) return;
    try {
      await fetch(`/api/providers/${channel.id}`, { method: 'DELETE' });
      setChannels((prev) => prev.filter((c) => c.id !== channel.id));
    } catch (error) {
      console.error('Failed to delete channel:', error);
    }
  };

  const columns: Column<Channel>[] = [
    {
      key: 'name',
      header: 'Channel Name',
      sortable: true,
      render: (row) => (
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-accent-primary/10 flex items-center justify-center">
            <Network className="w-5 h-5 text-accent-primary" />
          </div>
          <div>
            <p className="font-medium text-text-primary">{row.name}</p>
            <p className="text-xs text-text-muted">{row.provider}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'endpoint',
      header: 'Endpoint',
      render: (row) => (
        <code className="text-xs text-text-secondary bg-bg-elevated px-2 py-1 rounded">
          {row.endpoint}
        </code>
      ),
    },
    {
      key: 'priority',
      header: 'Priority',
      sortable: true,
      width: '100px',
      render: (row) => (
        <div className="flex items-center gap-2">
          <Gauge className="w-4 h-4 text-text-muted" />
          <span className="font-medium">#{row.priority}</span>
        </div>
      ),
    },
    {
      key: 'ratio',
      header: 'Ratio',
      sortable: true,
      width: '100px',
      render: (row) => (
        <div className="flex items-center gap-2">
          <div className="w-16 h-2 bg-bg-elevated rounded-full overflow-hidden">
            <div
              className="h-full bg-accent-primary rounded-full"
              style={{ width: `${row.ratio}%` }}
            />
          </div>
          <span className="text-sm text-text-secondary">{row.ratio}%</span>
        </div>
      ),
    },
    {
      key: 'is_active',
      header: 'Status',
      width: '100px',
      render: (row) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            toggleChannel(row);
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
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Channels</h1>
          <p className="text-sm text-text-tertiary mt-1">Manage upstream provider channels and routing</p>
        </div>
        <button
          onClick={() => {
            setEditingChannel(null);
            setShowModal(true);
          }}
          className="flex items-center gap-2 px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors"
        >
          <Plus className="w-4 h-4" />
          Add Channel
        </button>
      </div>

      {/* Load Balancing Info */}
      <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-accent-primary/10 flex items-center justify-center">
            <Network className="w-6 h-6 text-accent-primary" />
          </div>
          <div>
            <h3 className="text-sm font-medium text-text-primary">Load Balancing Configuration</h3>
            <p className="text-xs text-text-tertiary mt-0.5">
              Traffic is distributed based on priority and ratio settings. Active channels share load proportionally.
            </p>
          </div>
          <div className="ml-auto flex items-center gap-2">
            <span className="text-2xl font-bold text-accent-primary">{channels.filter((c) => c.is_active).length}</span>
            <span className="text-sm text-text-tertiary">Active</span>
          </div>
        </div>
      </div>

      {/* Channels Table */}
      <EnhancedTable
        columns={columns}
        data={channels}
        loading={loading}
        onEdit={(channel) => {
          setEditingChannel(channel);
          setShowModal(true);
        }}
        onDelete={deleteChannel}
        emptyMessage="No channels configured. Add your first channel to start routing traffic."
      />

      {/* Modal (Simplified) */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-bg-secondary border border-border-subtle rounded-xl p-6 w-full max-w-lg">
            <h2 className="text-lg font-semibold text-text-primary mb-4">
              {editingChannel ? 'Edit Channel' : 'Add New Channel'}
            </h2>
            <form className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Channel Name</label>
                <input
                  type="text"
                  defaultValue={editingChannel?.name || ''}
                  placeholder="e.g. OpenAI Global"
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:border-border-focus focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Provider</label>
                <select
                  defaultValue={editingChannel?.provider || 'openai'}
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                >
                  <option value="openai">OpenAI</option>
                  <option value="anthropic">Anthropic</option>
                  <option value="google">Google Gemini</option>
                  <option value="perplexity">Perplexity</option>
                  <option value="openrouter">OpenRouter</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">Endpoint</label>
                <input
                  type="text"
                  defaultValue={editingChannel?.endpoint || ''}
                  placeholder="https://api.openai.com/v1"
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:border-border-focus focus:outline-none"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1">Priority</label>
                  <input
                    type="number"
                    defaultValue={editingChannel?.priority || 1}
                    min={1}
                    max={100}
                    className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-text-secondary mb-1">Ratio (%)</label>
                  <input
                    type="number"
                    defaultValue={editingChannel?.ratio || 100}
                    min={0}
                    max={100}
                    className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 pt-4">
                <button
                  type="button"
                  onClick={() => setShowModal(false)}
                  className="px-4 py-2 text-text-secondary hover:text-text-primary transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  onClick={(e) => {
                    e.preventDefault();
                    setShowModal(false);
                    fetchChannels();
                  }}
                  className="px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors"
                >
                  {editingChannel ? 'Save Changes' : 'Add Channel'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
