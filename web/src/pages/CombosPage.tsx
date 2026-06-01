'use client';

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
  Zap,
  AlertTriangle,
  Settings,
  GitBranch,
} from 'lucide-react';
import { EnhancedTable, type Column } from '../tables/EnhancedTable';

// =============================================================================
// Type Definitions
// =============================================================================

interface Combo {
  id: string;
  name: string;
  description: string;
  strategy: RoutingStrategy;
  steps: RoutingChainStep[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

interface RoutingChainStep {
  id: string;
  order: number;
  provider_id: string;
  model_id: string;
  fallback_model?: string;
  weight: number;
  min_latency_ms?: number;
  max_cost_per_1k?: number;
  capabilities?: string[];
}

type RoutingStrategy =
  | 'priority'
  | 'fill_first'
  | 'least_used'
  | 'round_robin'
  | 'weighted'
  | 'p2c'
  | 'cost_optimized'
  | 'latency_optimized'
  | 'quality_first'
  | 'context_relay'
  | 'context_optimized'
  | 'random'
  | 'strict_random'
  | 'auto'
  | 'lkgp'
  | 'reset_aware';

interface RoutingStrategyInfo {
  id: RoutingStrategy;
  name: string;
  description: string;
  category: 'priority' | 'load_distribution' | 'optimization' | 'context_aware' | 'privacy' | 'smart';
}

// =============================================================================
// Routing Strategies Configuration
// =============================================================================

const ROUTING_STRATEGIES: RoutingStrategyInfo[] = [
  // Priority-based strategies
  { id: 'priority', name: 'Priority', description: 'Use highest priority available', category: 'priority' },
  { id: 'fill_first', name: 'Fill First', description: 'Drain subscription before paying', category: 'priority' },
  { id: 'least_used', name: 'Least Used', description: 'Use provider with fewest active requests', category: 'priority' },

  // Load distribution strategies
  { id: 'round_robin', name: 'Round Robin', description: 'Distribute evenly across providers', category: 'load_distribution' },
  { id: 'weighted', name: 'Weighted', description: 'Distribute by configured weight', category: 'load_distribution' },
  { id: 'p2c', name: 'P2C (Power of Two)', description: 'Pick least loaded from two random candidates', category: 'load_distribution' },

  // Optimization strategies
  { id: 'cost_optimized', name: 'Cost Optimized', description: 'Prefer cheapest viable option', category: 'optimization' },
  { id: 'latency_optimized', name: 'Latency Based', description: 'Route to fastest responding', category: 'optimization' },
  { id: 'quality_first', name: 'Quality First', description: 'Prefer highest quality model', category: 'optimization' },

  // Context-aware strategies
  { id: 'context_relay', name: 'Context Relay', description: 'Handoff with context summary', category: 'context_aware' },
  { id: 'context_optimized', name: 'Context Optimized', description: 'Optimize context window usage', category: 'context_aware' },

  // Privacy strategies
  { id: 'random', name: 'Random', description: 'Random selection', category: 'privacy' },
  { id: 'strict_random', name: 'Strict Random', description: 'Pure random for maximum privacy', category: 'privacy' },

  // Smart strategies
  { id: 'auto', name: 'Auto (9-Factor)', description: 'AI-powered 9-factor scoring', category: 'smart' },
  { id: 'lkgp', name: 'LKGP', description: 'Last Known Good Provider', category: 'smart' },
  { id: 'reset_aware', name: 'Reset Aware', description: 'Prioritize by quota reset time', category: 'smart' },
];

const STRATEGY_CATEGORY_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  priority: { bg: 'bg-blue-500/10', text: 'text-blue-400', border: 'border-blue-500/30' },
  load_distribution: { bg: 'bg-purple-500/10', text: 'text-purple-400', border: 'border-purple-500/30' },
  optimization: { bg: 'bg-green-500/10', text: 'text-green-400', border: 'border-green-500/30' },
  context_aware: { bg: 'bg-amber-500/10', text: 'text-amber-400', border: 'border-amber-500/30' },
  privacy: { bg: 'bg-cyan-500/10', text: 'text-cyan-400', border: 'border-cyan-500/30' },
  smart: { bg: 'bg-pink-500/10', text: 'text-pink-400', border: 'border-pink-500/30' },
};

// =============================================================================
// Provider Mock Data
// =============================================================================

const MOCK_PROVIDERS = [
  { id: 'openai', name: 'OpenAI', models: ['gpt-4o', 'gpt-4', 'gpt-3.5-turbo'] },
  { id: 'anthropic', name: 'Anthropic', models: ['claude-3-5-sonnet', 'claude-3-opus', 'claude-3-haiku'] },
  { id: 'google', name: 'Google', models: ['gemini-1.5-pro', 'gemini-1.5-flash', 'gemini-pro'] },
  { id: 'deepseek', name: 'DeepSeek', models: ['deepseek-chat', 'deepseek-coder'] },
  { id: 'groq', name: 'Groq', models: ['llama-3.1-70b', 'mixtral-8x7b'] },
];

// =============================================================================
// CombosPage Component
// =============================================================================

export function CombosPage() {
  const [combos, setCombos] = useState<Combo[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editingCombo, setEditingCombo] = useState<Combo | null>(null);
  const [showStrategiesPanel, setShowStrategiesPanel] = useState(false);

  // New/Edit combo form state
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    strategy: 'priority' as RoutingStrategy,
    steps: [] as RoutingChainStep[],
  });

  useEffect(() => {
    fetchCombos();
  }, []);

  const fetchCombos = async () => {
    try {
      const res = await fetch('/api/combos');
      if (res.ok) {
        const data = await res.json();
        setCombos(data.combos || []);
      }
    } catch {
      // Use mock data for development
      setCombos([
        {
          id: 'combo-1',
          name: 'Production Primary',
          description: 'Primary production routing with fallback',
          strategy: 'priority',
          steps: [
            { id: 'step-1', order: 1, provider_id: 'openai', model_id: 'gpt-4o', weight: 1.0 },
            { id: 'step-2', order: 2, provider_id: 'anthropic', model_id: 'claude-3-5-sonnet', weight: 1.0 },
          ],
          is_active: true,
          created_at: '2024-01-15T10:00:00Z',
          updated_at: '2024-01-20T14:30:00Z',
        },
        {
          id: 'combo-2',
          name: 'Cost-Optimized',
          description: 'Route to cheapest viable option',
          strategy: 'cost_optimized',
          steps: [
            { id: 'step-3', order: 1, provider_id: 'deepseek', model_id: 'deepseek-chat', weight: 0.6 },
            { id: 'step-4', order: 2, provider_id: 'groq', model_id: 'llama-3.1-70b', weight: 0.4 },
          ],
          is_active: true,
          created_at: '2024-01-18T09:00:00Z',
          updated_at: '2024-01-22T11:00:00Z',
        },
        {
          id: 'combo-3',
          name: 'Smart Auto-Routing',
          description: 'AI-powered 9-factor scoring',
          strategy: 'auto',
          steps: [
            { id: 'step-5', order: 1, provider_id: 'openai', model_id: 'gpt-4o', weight: 0.33 },
            { id: 'step-6', order: 2, provider_id: 'anthropic', model_id: 'claude-3-5-sonnet', weight: 0.33 },
            { id: 'step-7', order: 3, provider_id: 'google', model_id: 'gemini-1.5-pro', weight: 0.34 },
          ],
          is_active: false,
          created_at: '2024-01-20T16:00:00Z',
          updated_at: '2024-01-21T10:00:00Z',
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const toggleCombo = async (combo: Combo) => {
    const updated = { ...combo, is_active: !combo.is_active };
    try {
      await fetch(`/api/combos/${combo.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ is_active: updated.is_active }),
      });
      setCombos((prev) => prev.map((c) => (c.id === combo.id ? updated : c)));
    } catch (error) {
      console.error('Failed to toggle combo:', error);
    }
  };

  const deleteCombo = async (combo: Combo) => {
    if (!confirm(`Delete combo "${combo.name}"?`)) return;
    try {
      await fetch(`/api/combos/${combo.id}`, { method: 'DELETE' });
      setCombos((prev) => prev.filter((c) => c.id !== combo.id));
    } catch (error) {
      console.error('Failed to delete combo:', error);
    }
  };

  const openCreateModal = () => {
    setEditingCombo(null);
    setFormData({ name: '', description: '', strategy: 'priority', steps: [] });
    setShowModal(true);
  };

  const openEditModal = (combo: Combo) => {
    setEditingCombo(combo);
    setFormData({
      name: combo.name,
      description: combo.description,
      strategy: combo.strategy,
      steps: [...combo.steps],
    });
    setShowModal(true);
  };

  const handleCreateOrUpdate = (e: React.FormEvent) => {
    e.preventDefault();
    setShowModal(false);
    fetchCombos();
  };

  const getStrategyInfo = (strategy: RoutingStrategy) => {
    return ROUTING_STRATEGIES.find((s) => s.id === strategy) || {
      id: strategy,
      name: strategy,
      description: 'Unknown strategy',
      category: 'priority',
    };
  };

  const getCategoryStyle = (category: string) => {
    return STRATEGY_CATEGORY_COLORS[category] || STRATEGY_CATEGORY_COLORS.priority;
  };

  // =============================================================================
  // Table Columns
  // =============================================================================

  const columns: Column<Combo>[] = [
    {
      key: 'name',
      header: 'Combo Name',
      sortable: true,
      render: (row) => (
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-lg bg-accent-primary/10 flex items-center justify-center">
            <GitBranch className="w-5 h-5 text-accent-primary" />
          </div>
          <div>
            <p className="font-medium text-text-primary">{row.name}</p>
            <p className="text-xs text-text-muted">{row.description}</p>
          </div>
        </div>
      ),
    },
    {
      key: 'strategy',
      header: 'Strategy',
      sortable: true,
      width: '160px',
      render: (row) => {
        const strategy = getStrategyInfo(row.strategy);
        const style = getCategoryStyle(strategy.category);
        return (
          <span
            className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${style.bg} ${style.text} ${style.border}`}
          >
            <Zap className="w-3 h-3" />
            {strategy.name}
          </span>
        );
      },
    },
    {
      key: 'steps',
      header: 'Providers',
      render: (row) => (
        <div className="flex flex-wrap gap-1">
          {row.steps.slice(0, 3).map((step, i) => {
            const provider = MOCK_PROVIDERS.find((p) => p.id === step.provider_id);
            return (
              <span
                key={step.id}
                className="inline-flex items-center gap-1 px-2 py-0.5 bg-bg-elevated rounded text-xs text-text-secondary"
              >
                {provider?.name || step.provider_id}
                {i < row.steps.length - 1 && i < 2 && (
                  <span className="text-text-muted ml-1">→</span>
                )}
              </span>
            );
          })}
          {row.steps.length > 3 && (
            <span className="px-2 py-0.5 bg-bg-elevated rounded text-xs text-text-muted">
              +{row.steps.length - 3}
            </span>
          )}
        </div>
      ),
    },
    {
      key: 'is_active',
      header: 'Status',
      width: '120px',
      render: (row) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            toggleCombo(row);
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

  // =============================================================================
  // Render
  // =============================================================================

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Combos & Routing</h1>
          <p className="text-sm text-text-tertiary mt-1">
            Manage routing strategies and provider combinations
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => setShowStrategiesPanel(!showStrategiesPanel)}
            className={`flex items-center gap-2 px-4 py-2 rounded-lg border transition-colors ${
              showStrategiesPanel
                ? 'bg-accent-primary/10 border-accent-primary text-accent-primary'
                : 'bg-bg-tertiary border-border-subtle text-text-secondary hover:text-text-primary'
            }`}
          >
            <Settings className="w-4 h-4" />
            {showStrategiesPanel ? 'Hide' : 'View'} Strategies
          </button>
          <button
            onClick={openCreateModal}
            className="flex items-center gap-2 px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors"
          >
            <Plus className="w-4 h-4" />
            Create Combo
          </button>
        </div>
      </div>

      {/* Stats Row */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-accent-primary/10 flex items-center justify-center">
              <GitBranch className="w-5 h-5 text-accent-primary" />
            </div>
            <div>
              <p className="text-2xl font-bold text-text-primary">{combos.length}</p>
              <p className="text-xs text-text-tertiary">Total Combos</p>
            </div>
          </div>
        </div>
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-success/10 flex items-center justify-center">
              <ToggleRight className="w-5 h-5 text-success" />
            </div>
            <div>
              <p className="text-2xl font-bold text-success">{combos.filter((c) => c.is_active).length}</p>
              <p className="text-xs text-text-tertiary">Active</p>
            </div>
          </div>
        </div>
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-purple-500/10 flex items-center justify-center">
              <Zap className="w-5 h-5 text-purple-400" />
            </div>
            <div>
              <p className="text-2xl font-bold text-text-primary">{ROUTING_STRATEGIES.length}</p>
              <p className="text-xs text-text-tertiary">Strategies</p>
            </div>
          </div>
        </div>
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <div className="flex items-center gap-3">
            <div className="w-10 h-10 rounded-lg bg-blue-500/10 flex items-center justify-center">
              <Network className="w-5 h-5 text-blue-400" />
            </div>
            <div>
              <p className="text-2xl font-bold text-text-primary">{MOCK_PROVIDERS.length}</p>
              <p className="text-xs text-text-tertiary">Providers</p>
            </div>
          </div>
        </div>
      </div>

      {/* Strategies Panel */}
      {showStrategiesPanel && (
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <h3 className="text-sm font-medium text-text-primary mb-4 flex items-center gap-2">
            <Zap className="w-4 h-4 text-accent-primary" />
            Available Routing Strategies ({ROUTING_STRATEGIES.length})
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {ROUTING_STRATEGIES.map((strategy) => {
              const style = getCategoryStyle(strategy.category);
              return (
                <div
                  key={strategy.id}
                  className={`flex items-start gap-3 p-3 rounded-lg bg-bg-secondary border ${style.border}`}
                >
                  <div className={`p-2 rounded-md ${style.bg}`}>
                    <Zap className={`w-4 h-4 ${style.text}`} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="font-medium text-text-primary">{strategy.name}</span>
                      <span className={`text-2xs px-1.5 py-0.5 rounded ${style.bg} ${style.text}`}>
                        {strategy.category.replace('_', ' ')}
                      </span>
                    </div>
                    <p className="text-xs text-text-muted mt-0.5">{strategy.description}</p>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}

      {/* Combos Table */}
      <EnhancedTable
        columns={columns}
        data={combos}
        loading={loading}
        onEdit={openEditModal}
        onDelete={deleteCombo}
        emptyMessage="No combos configured. Create your first combo to start routing traffic."
      />

      {/* Create/Edit Modal */}
      {showModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-bg-secondary border border-border-subtle rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <h2 className="text-lg font-semibold text-text-primary mb-4">
              {editingCombo ? 'Edit Combo' : 'Create New Combo'}
            </h2>
            <form onSubmit={handleCreateOrUpdate} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">
                  Combo Name
                </label>
                <input
                  type="text"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="e.g., Production Primary"
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:border-border-focus focus:outline-none"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">
                  Description
                </label>
                <input
                  type="text"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="Optional description"
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary placeholder:text-text-muted focus:border-border-focus focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-1">
                  Routing Strategy
                </label>
                <select
                  value={formData.strategy}
                  onChange={(e) => setFormData({ ...formData, strategy: e.target.value as RoutingStrategy })}
                  className="w-full px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary focus:border-border-focus focus:outline-none"
                >
                  {ROUTING_STRATEGIES.map((s) => (
                    <option key={s.id} value={s.id}>
                      {s.name} - {s.description}
                    </option>
                  ))}
                </select>
              </div>

              {/* Provider Steps */}
              <div>
                <label className="block text-sm font-medium text-text-secondary mb-2">
                  Provider Steps
                </label>
                <div className="space-y-2">
                  {formData.steps.map((step, index) => {
                    const provider = MOCK_PROVIDERS.find((p) => p.id === step.provider_id);
                    return (
                      <div
                        key={step.id}
                        className="flex items-center gap-2 p-2 bg-bg-tertiary rounded-lg"
                      >
                        <span className="w-6 h-6 rounded-full bg-accent-primary/10 flex items-center justify-center text-xs text-accent-primary font-medium">
                          {step.order}
                        </span>
                        <select
                          value={step.provider_id}
                          onChange={(e) => {
                            const newSteps = [...formData.steps];
                            newSteps[index].provider_id = e.target.value;
                            setFormData({ ...formData, steps: newSteps });
                          }}
                          className="flex-1 px-2 py-1 bg-bg-secondary border border-border-subtle rounded text-text-primary text-sm"
                        >
                          {MOCK_PROVIDERS.map((p) => (
                            <option key={p.id} value={p.id}>
                              {p.name}
                            </option>
                          ))}
                        </select>
                        <select
                          value={step.model_id}
                          onChange={(e) => {
                            const newSteps = [...formData.steps];
                            newSteps[index].model_id = e.target.value;
                            setFormData({ ...formData, steps: newSteps });
                          }}
                          className="flex-1 px-2 py-1 bg-bg-secondary border border-border-subtle rounded text-text-primary text-sm"
                        >
                          {provider?.models.map((m) => (
                            <option key={m} value={m}>
                              {m}
                            </option>
                          ))}
                        </select>
                        <button
                          type="button"
                          onClick={() => {
                            const newSteps = formData.steps.filter((_, i) => i !== index);
                            setFormData({ ...formData, steps: newSteps });
                          }}
                          className="p-1 text-text-muted hover:text-error transition-colors"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      </div>
                    );
                  })}
                  <button
                    type="button"
                    onClick={() => {
                      const newStep: RoutingChainStep = {
                        id: `step-${Date.now()}`,
                        order: formData.steps.length + 1,
                        provider_id: MOCK_PROVIDERS[0].id,
                        model_id: MOCK_PROVIDERS[0].models[0],
                        weight: 1.0,
                      };
                      setFormData({ ...formData, steps: [...formData.steps, newStep] });
                    }}
                    className="w-full flex items-center justify-center gap-2 px-3 py-2 border border-dashed border-border-subtle rounded-lg text-sm text-text-muted hover:text-text-secondary hover:border-border-default transition-colors"
                  >
                    <Plus className="w-4 h-4" />
                    Add Provider Step
                  </button>
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
                  disabled={!formData.name || formData.steps.length === 0}
                  className="px-4 py-2 bg-accent-primary text-white rounded-lg hover:bg-accent-hover transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {editingCombo ? 'Save Changes' : 'Create Combo'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

export default CombosPage;
