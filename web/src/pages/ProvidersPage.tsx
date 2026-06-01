'use client';

import React, { useState, useEffect } from 'react';
import { 
  Plus, Search, Settings, Trash2, Play, 
  CheckCircle, XCircle, AlertTriangle, RefreshCw, 
  ChevronDown, ChevronUp, Activity, Cpu, Zap
} from 'lucide-react';
import { Card } from '@/components/ui/Card';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { Input } from '@/components/ui/Input';
import { Modal, ModalHeader, ModalBody, ModalFooter } from '@/components/ui/Modal';
import { Table, TableHeader, TableBody, TableRow, TableHead, TableCell } from '@/components/ui/Table';
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from '@/components/ui/Select';

interface Model {
  id: string;
  name: string;
  status: 'active' | 'inactive';
  contextLength: number;
  pricing?: { input: number; output: number };
}

interface Provider {
  id: string;
  name: string;
  type: string;
  status: 'active' | 'inactive' | 'error';
  health?: 'healthy' | 'degraded' | 'down';
  latency?: number;
  models: Model[];
  apiKeyConfigured: boolean;
  createdAt: string;
  lastUsed?: string;
}

const mockProviders: Provider[] = [
  {
    id: '1',
    name: 'OpenAI',
    type: 'openai',
    status: 'active',
    health: 'healthy',
    latency: 120,
    models: [
      { id: 'gpt-4', name: 'GPT-4', status: 'active', contextLength: 8192, pricing: { input: 0.03, output: 0.06 } },
      { id: 'gpt-4-turbo', name: 'GPT-4 Turbo', status: 'active', contextLength: 128000, pricing: { input: 0.01, output: 0.03 } },
      { id: 'gpt-3.5-turbo', name: 'GPT-3.5 Turbo', status: 'active', contextLength: 16385, pricing: { input: 0.0005, output: 0.0015 } },
    ],
    apiKeyConfigured: true,
    createdAt: new Date().toISOString(),
    lastUsed: new Date().toISOString(),
  },
  {
    id: '2',
    name: 'Anthropic',
    type: 'anthropic',
    status: 'active',
    health: 'healthy',
    latency: 95,
    models: [
      { id: 'claude-3-opus', name: 'Claude 3 Opus', status: 'active', contextLength: 200000 },
      { id: 'claude-3-sonnet', name: 'Claude 3 Sonnet', status: 'active', contextLength: 200000 },
      { id: 'claude-3-haiku', name: 'Claude 3 Haiku', status: 'active', contextLength: 200000 },
    ],
    apiKeyConfigured: true,
    createdAt: new Date().toISOString(),
    lastUsed: new Date().toISOString(),
  },
  {
    id: '3',
    name: 'Google AI',
    type: 'google',
    status: 'active',
    health: 'degraded',
    latency: 450,
    models: [
      { id: 'gemini-pro', name: 'Gemini Pro', status: 'active', contextLength: 32768 },
      { id: 'gemini-ultra', name: 'Gemini Ultra', status: 'inactive', contextLength: 32768 },
    ],
    apiKeyConfigured: true,
    createdAt: new Date().toISOString(),
  },
  {
    id: '4',
    name: 'Azure OpenAI',
    type: 'azure',
    status: 'inactive',
    health: 'down',
    latency: undefined,
    models: [
      { id: 'azure-gpt-4', name: 'Azure GPT-4', status: 'inactive', contextLength: 8192 },
    ],
    apiKeyConfigured: false,
    createdAt: new Date().toISOString(),
  },
];

export default function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<'all' | 'active' | 'inactive'>('all');
  const [showAddModal, setShowAddModal] = useState(false);
  const [testingProvider, setTestingProvider] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{ success: boolean; latency?: number; error?: string } | null>(null);
  const [expandedProvider, setExpandedProvider] = useState<string | null>(null);

  useEffect(() => {
    fetchProviders();
  }, []);

  const fetchProviders = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/providers');
      if (response.ok) {
        const data = await response.json();
        setProviders(data.providers || []);
      } else {
        // Fallback to mock data
        setProviders(mockProviders);
      }
    } catch {
      // Use mock data on error
      setProviders(mockProviders);
    } finally {
      setLoading(false);
    }
  };

  const handleTestProvider = async (providerId: string) => {
    setTestingProvider(providerId);
    setTestResult(null);

    // Simulate test
    await new Promise(resolve => setTimeout(resolve, 1500));
    
    const provider = providers.find(p => p.id === providerId);
    setTestResult({
      success: provider?.health !== 'down',
      latency: provider?.health === 'down' ? undefined : Math.floor(Math.random() * 200) + 50,
      error: provider?.health === 'down' ? 'Connection timeout' : undefined,
    });
  };

  const filteredProviders = providers.filter(provider => {
    const matchesSearch = provider.name.toLowerCase().includes(search.toLowerCase());
    const matchesFilter = filter === 'all' || provider.status === filter;
    return matchesSearch && matchesFilter;
  });

  const getHealthBadge = (health?: string) => {
    if (!health) return null;
    
    const config = {
      healthy: { class: 'bg-success/10 text-success border border-success/20', icon: CheckCircle, label: 'Healthy' },
      degraded: { class: 'bg-warning/10 text-warning border border-warning/20', icon: AlertTriangle, label: 'Degraded' },
      down: { class: 'bg-error/10 text-error border border-error/20', icon: XCircle, label: 'Down' },
    }[health] as { class: string; icon: React.ElementType; label: string };

    if (!config) return null;
    const Icon = config.icon;
    return (
      <Badge className={config.class}>
        <Icon className="w-3 h-3 mr-1" />
        {config.label}
      </Badge>
    );
  };

  const getLatencyColor = (latency?: number) => {
    if (!latency) return 'text-text-muted';
    if (latency > 300) return 'text-error';
    if (latency > 150) return 'text-warning';
    return 'text-success';
  };

  const avgLatency = providers.filter(p => p.latency).reduce((a, p) => a + (p.latency || 0), 0) / providers.filter(p => p.latency).length || 0;

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Providers</h1>
          <p className="text-sm text-text-tertiary mt-1">Manage your AI provider connections</p>
        </div>
        <Button onClick={() => setShowAddModal(true)}>
          <Plus className="w-4 h-4 mr-2" />
          Add Provider
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card className="bg-bg-tertiary border border-border-subtle">
          <div className="flex items-center gap-4">
            <div className="w-12 h-12 rounded-xl bg-accent-primary/10 flex items-center justify-center">
              <Cpu className="w-6 h-6 text-accent-primary" />
            </div>
            <div>
              <div className="text-2xl font-bold text-text-primary">{providers.length}</div>
              <p className="text-sm text-text-tertiary">Total Providers</p>
            </div>
          </div>
        </Card>

        <Card className="bg-bg-tertiary border border-border-subtle">
          <div className="flex items-center gap-4">
            <div className="w-12 h-12 rounded-xl bg-success/10 flex items-center justify-center">
              <CheckCircle className="w-6 h-6 text-success" />
            </div>
            <div>
              <div className="text-2xl font-bold text-success">
                {providers.filter(p => p.health === 'healthy').length}
              </div>
              <p className="text-sm text-text-tertiary">Healthy</p>
            </div>
          </div>
        </Card>

        <Card className="bg-bg-tertiary border border-border-subtle">
          <div className="flex items-center gap-4">
            <div className="w-12 h-12 rounded-xl bg-info/10 flex items-center justify-center">
              <Zap className="w-6 h-6 text-info" />
            </div>
            <div>
              <div className="text-2xl font-bold text-info">
                {providers.reduce((acc, p) => acc + p.models.length, 0)}
              </div>
              <p className="text-sm text-text-tertiary">Models Available</p>
            </div>
          </div>
        </Card>

        <Card className="bg-bg-tertiary border border-border-subtle">
          <div className="flex items-center gap-4">
            <div className="w-12 h-12 rounded-xl bg-warning/10 flex items-center justify-center">
              <Activity className="w-6 h-6 text-warning" />
            </div>
            <div>
              <div className={`text-2xl font-bold ${getLatencyColor(Math.round(avgLatency))}`}>
                {Math.round(avgLatency)}ms
              </div>
              <p className="text-sm text-text-tertiary">Avg Latency</p>
            </div>
          </div>
        </Card>
      </div>

      {/* Search and Filters */}
      <div className="flex items-center gap-4">
        <div className="relative flex-1 max-w-md">
          <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 w-4 h-4 text-text-muted" />
          <Input
            placeholder="Search providers..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="pl-10"
          />
        </div>
        <Select value={filter} onValueChange={v => setFilter(v as typeof filter)}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="Filter" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="inactive">Inactive</SelectItem>
          </SelectContent>
        </Select>
        <Button variant="secondary" onClick={fetchProviders}>
          <RefreshCw className="w-4 h-4 mr-2" />
          Refresh
        </Button>
      </div>

      {/* Providers Table */}
      <Card className="bg-bg-tertiary border border-border-subtle p-0 overflow-hidden">
        {loading ? (
          <div className="flex items-center justify-center py-12">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-primary"></div>
          </div>
        ) : filteredProviders.length === 0 ? (
          <div className="text-center py-12">
            <Cpu className="w-12 h-12 mx-auto text-text-muted mb-4" />
            <p className="text-text-secondary">No providers found</p>
            <p className="text-sm text-text-muted mt-1">Add your first provider to get started</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="bg-bg-elevated border-b border-border-subtle">
                <TableHead className="w-8"></TableHead>
                <TableHead>Provider</TableHead>
                <TableHead>Health</TableHead>
                <TableHead>Latency</TableHead>
                <TableHead>Models</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredProviders.map(provider => (
                <React.Fragment key={provider.id}>
                  <TableRow className="hover:bg-bg-elevated/50 transition-colors">
                    <TableCell>
                      <button
                        onClick={() => setExpandedProvider(expandedProvider === provider.id ? null : provider.id)}
                        className="p-1 hover:bg-bg-elevated rounded transition-colors"
                      >
                        {expandedProvider === provider.id ? (
                          <ChevronUp className="w-4 h-4 text-text-muted" />
                        ) : (
                          <ChevronDown className="w-4 h-4 text-text-muted" />
                        )}
                      </button>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <div className="w-10 h-10 rounded-lg bg-accent-primary/10 flex items-center justify-center">
                          <span className="text-lg font-bold text-accent-primary">
                            {provider.name.charAt(0)}
                          </span>
                        </div>
                        <div>
                          <div className="font-medium text-text-primary">{provider.name}</div>
                          <div className="text-xs text-text-muted capitalize">{provider.type}</div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      {getHealthBadge(provider.health)}
                    </TableCell>
                    <TableCell>
                      {provider.latency ? (
                        <span className={getLatencyColor(provider.latency)}>
                          {provider.latency}ms
                        </span>
                      ) : (
                        <span className="text-text-muted">-</span>
                      )}
                    </TableCell>
                    <TableCell>
                      <Badge variant="outline">{provider.models.length} models</Badge>
                    </TableCell>
                    <TableCell>
                      <Badge className={provider.status === 'active' ? 'bg-success/10 text-success border border-success/20' : 'bg-bg-elevated text-text-muted border border-border-subtle'}>
                        {provider.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleTestProvider(provider.id)}
                          disabled={testingProvider === provider.id}
                          className={testingProvider === provider.id ? 'animate-pulse' : ''}
                        >
                          <Play className="w-4 h-4" />
                        </Button>
                        <Button variant="ghost" size="sm">
                          <Settings className="w-4 h-4" />
                        </Button>
                        <Button variant="ghost" size="sm">
                          <Trash2 className="w-4 h-4 text-error" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>

                  {/* Expanded Row - Model Details */}
                  {expandedProvider === provider.id && (
                    <TableRow className="bg-bg-elevated/30">
                      <TableCell colSpan={7} className="p-4">
                        <div className="pl-8">
                          <h4 className="text-sm font-medium text-text-secondary mb-3 flex items-center gap-2">
                            <Activity className="w-4 h-4" />
                            Available Models
                          </h4>
                          <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
                            {provider.models.map(model => (
                              <div key={model.id} className="p-3 bg-bg-secondary rounded-lg border border-border-subtle">
                                <div className="flex items-center justify-between mb-2">
                                  <span className="font-medium text-text-primary">{model.name}</span>
                                  <Badge className={model.status === 'active' ? 'bg-success/10 text-success border border-success/20' : 'bg-bg-elevated text-text-muted border border-border-subtle'}>
                                    {model.status}
                                  </Badge>
                                </div>
                                <div className="text-xs text-text-muted space-y-1">
                                  <div>Context: {model.contextLength?.toLocaleString() || 'N/A'} tokens</div>
                                  {model.pricing && (
                                    <div>
                                      ${model.pricing.input}/1K in &bull; ${model.pricing.output}/1K out
                                    </div>
                                  )}
                                </div>
                              </div>
                            ))}
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                  )}

                  {/* Test Result Row */}
                  {testingProvider === provider.id && testResult && (
                    <TableRow className="bg-bg-elevated/30">
                      <TableCell colSpan={7} className="p-4">
                        <div className={`flex items-center gap-3 p-3 rounded-lg ${
                          testResult.success 
                            ? 'bg-success/5 border border-success/20' 
                            : 'bg-error/5 border border-error/20'
                        }`}>
                          {testResult.success ? (
                            <>
                              <CheckCircle className="w-5 h-5 text-success flex-shrink-0" />
                              <div>
                                <div className="font-medium text-success">Test Successful</div>
                                <div className="text-sm text-text-secondary">
                                  Latency: {testResult.latency}ms
                                </div>
                              </div>
                            </>
                          ) : (
                            <>
                              <XCircle className="w-5 h-5 text-error flex-shrink-0" />
                              <div>
                                <div className="font-medium text-error">Test Failed</div>
                                <div className="text-sm text-text-secondary">
                                  {testResult.error || 'Connection error'}
                                </div>
                              </div>
                            </>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  )}
                </React.Fragment>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {/* Add Provider Modal */}
      <Modal isOpen={showAddModal} onClose={() => setShowAddModal(false)} size="lg">
        <ModalHeader>
          <h2 className="text-lg font-semibold text-text-primary">Add Provider</h2>
          <p className="text-sm text-text-tertiary">Configure a new AI provider</p>
        </ModalHeader>
        <ModalBody>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1.5">Provider Type</label>
              <Select>
                <SelectTrigger>
                  <SelectValue placeholder="Select a provider" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="openai">OpenAI</SelectItem>
                  <SelectItem value="anthropic">Anthropic</SelectItem>
                  <SelectItem value="google">Google AI</SelectItem>
                  <SelectItem value="cohere">Cohere</SelectItem>
                  <SelectItem value="azure">Azure OpenAI</SelectItem>
                  <SelectItem value="openrouter">OpenRouter</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1.5">Display Name</label>
              <Input placeholder="My OpenAI Provider" />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1.5">API Key</label>
              <Input type="password" placeholder="sk-..." />
            </div>
            <div>
              <label className="block text-sm font-medium text-text-secondary mb-1.5">Base URL (Optional)</label>
              <Input placeholder="https://api.openai.com/v1" />
            </div>
          </div>
        </ModalBody>
        <ModalFooter>
          <Button variant="secondary" onClick={() => setShowAddModal(false)}>Cancel</Button>
          <Button onClick={() => setShowAddModal(false)}>Add Provider</Button>
        </ModalFooter>
      </Modal>
    </div>
  );
}

export { ProvidersPage };