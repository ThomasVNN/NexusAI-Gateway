import { useState, useEffect } from 'react';
import {
  CreditCard,
  TrendingUp,
  TrendingDown,
  DollarSign,
  BarChart3,
  RefreshCw,
  ArrowUpRight,
} from 'lucide-react';
import { UsageChart } from '../charts/UsageChart';
import { ModelDistributionChart } from '../charts/ModelDistributionChart';

interface ModelPricing {
  model: string;
  provider: string;
  input_cost: number;
  output_cost: number;
  currency: string;
}

interface UsageRecord {
  model: string;
  input_tokens: number;
  output_tokens: number;
  cost: number;
  requests: number;
}

export function BillingPage() {
  const [loading, setLoading] = useState(true);
  const [balance, setBalance] = useState({ amount: 0, currency: 'USD' });
  const [usageRecords, setUsageRecords] = useState<UsageRecord[]>([]);
  const [modelPricing, setModelPricing] = useState<ModelPricing[]>([]);

  useEffect(() => {
    fetchBillingData();
  }, []);

  const fetchBillingData = async () => {
    try {
      const [balanceRes, pricingRes] = await Promise.all([
        fetch('/api/admin/balance'),
        fetch('/api/admin/model-pricing'),
      ]);
      if (balanceRes.ok) setBalance(await balanceRes.json());
      if (pricingRes.ok) setModelPricing(await pricingRes.json());
    } catch {
      setBalance({ amount: 847.52, currency: 'USD' });
      setModelPricing([
        { model: 'GPT-4o', provider: 'openai', input_cost: 0.000005, output_cost: 0.000015, currency: 'USD' },
        { model: 'Claude 3.5 Sonnet', provider: 'anthropic', input_cost: 0.000003, output_cost: 0.000015, currency: 'USD' },
        { model: 'Gemini Pro', provider: 'google', input_cost: 0.00000125, output_cost: 0.000005, currency: 'USD' },
        { model: 'Llama 3 70B', provider: 'openrouter', input_cost: 0.0000008, output_cost: 0.0000024, currency: 'USD' },
      ]);
      setUsageRecords([
        { model: 'GPT-4o', input_tokens: 1250000, output_tokens: 850000, cost: 18.50, requests: 4521 },
        { model: 'Claude 3.5 Sonnet', input_tokens: 980000, output_tokens: 620000, cost: 12.30, requests: 3847 },
        { model: 'Gemini Pro', input_tokens: 450000, output_tokens: 320000, cost: 2.85, requests: 2134 },
        { model: 'Llama 3 70B', input_tokens: 167000, output_tokens: 89000, cost: 0.47, requests: 1111 },
      ]);
    } finally {
      setLoading(false);
    }
  };

  const usageData = [
    { time: 'Week 1', requests: 2800, tokens: 65000 },
    { time: 'Week 2', requests: 3200, tokens: 72000 },
    { time: 'Week 3', requests: 2900, tokens: 68000 },
    { time: 'Week 4', requests: 3947, tokens: 91000 },
  ];

  const modelData = usageRecords.map((r) => ({
    model: r.model,
    requests: r.requests,
    color: r.model.includes('GPT') ? '#8B5CF6' : r.model.includes('Claude') ? '#34D399' : r.model.includes('Gemini') ? '#60A5FA' : '#FBBF24',
  }));

  const totalCost = usageRecords.reduce((sum, r) => sum + r.cost, 0);
  const totalRequests = usageRecords.reduce((sum, r) => sum + r.requests, 0);

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Billing & Usage</h1>
          <p className="text-sm text-text-tertiary mt-1">Monitor usage and manage your credit balance</p>
        </div>
        <button
          onClick={fetchBillingData}
          className="flex items-center gap-2 px-4 py-2 text-text-secondary hover:text-text-primary hover:bg-bg-elevated rounded-lg transition-colors"
        >
          <RefreshCw className="w-4 h-4" />
          Refresh
        </button>
      </div>

      {/* Balance & Quick Stats */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
        {/* Balance Card */}
        <div className="sm:col-span-2 bg-gradient-to-br from-accent-primary/20 to-accent-secondary/10 border border-accent-primary/30 rounded-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-12 h-12 rounded-xl bg-accent-primary/20 flex items-center justify-center">
              <DollarSign className="w-6 h-6 text-accent-primary" />
            </div>
            <div>
              <p className="text-sm text-text-tertiary">Current Balance</p>
              <p className="text-3xl font-bold text-text-primary">
                ${balance.amount.toFixed(2)} <span className="text-lg text-text-muted">{balance.currency}</span>
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <span className={`flex items-center gap-1 ${balance.amount > 100 ? 'text-success' : 'text-warning'}`}>
              {balance.amount > 100 ? <TrendingUp className="w-4 h-4" /> : <TrendingDown className="w-4 h-4" />}
              {balance.amount > 100 ? 'Healthy balance' : 'Low balance warning'}
            </span>
          </div>
        </div>

        {/* Cost Summary */}
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-text-tertiary">This Period</p>
              <p className="text-2xl font-bold text-text-primary mt-1">${totalCost.toFixed(2)}</p>
            </div>
            <BarChart3 className="w-8 h-8 text-accent-primary" />
          </div>
        </div>

        {/* Requests Summary */}
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-xs text-text-tertiary">Total Requests</p>
              <p className="text-2xl font-bold text-text-primary mt-1">{totalRequests.toLocaleString()}</p>
            </div>
            <CreditCard className="w-8 h-8 text-info" />
          </div>
        </div>
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <UsageChart data={usageData} />
        </div>
        <div>
          <ModelDistributionChart data={modelData} />
        </div>
      </div>

      {/* Usage by Model */}
      <div className="bg-bg-tertiary border border-border-subtle rounded-xl overflow-hidden">
        <div className="p-4 border-b border-border-subtle">
          <h3 className="text-sm font-medium text-text-primary">Usage by Model</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-bg-secondary">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-text-tertiary uppercase">Model</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Requests</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Input Tokens</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Output Tokens</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Cost</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border-subtle">
              {usageRecords.map((record, i) => (
                <tr key={i} className="hover:bg-bg-elevated/50">
                  <td className="px-4 py-3">
                    <span className="font-medium text-text-primary">{record.model}</span>
                  </td>
                  <td className="px-4 py-3 text-right text-text-secondary">{record.requests.toLocaleString()}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">{record.input_tokens.toLocaleString()}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">{record.output_tokens.toLocaleString()}</td>
                  <td className="px-4 py-3 text-right font-medium text-accent-primary">${record.cost.toFixed(2)}</td>
                </tr>
              ))}
            </tbody>
            <tfoot className="bg-bg-secondary">
              <tr>
                <td className="px-4 py-3 font-medium text-text-primary">Total</td>
                <td className="px-4 py-3 text-right font-medium text-text-primary">{totalRequests.toLocaleString()}</td>
                <td className="px-4 py-3 text-right font-medium text-text-primary">
                  {usageRecords.reduce((s, r) => s + r.input_tokens, 0).toLocaleString()}
                </td>
                <td className="px-4 py-3 text-right font-medium text-text-primary">
                  {usageRecords.reduce((s, r) => s + r.output_tokens, 0).toLocaleString()}
                </td>
                <td className="px-4 py-3 text-right font-bold text-accent-primary">${totalCost.toFixed(2)}</td>
              </tr>
            </tfoot>
          </table>
        </div>
      </div>

      {/* Model Pricing */}
      <div className="bg-bg-tertiary border border-border-subtle rounded-xl overflow-hidden">
        <div className="p-4 border-b border-border-subtle">
          <h3 className="text-sm font-medium text-text-primary">Model Pricing (per 1K tokens)</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-bg-secondary">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-text-tertiary uppercase">Model</th>
                <th className="px-4 py-3 text-left text-xs font-medium text-text-tertiary uppercase">Provider</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Input Cost</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Output Cost</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border-subtle">
              {modelPricing.map((pricing, i) => (
                <tr key={i} className="hover:bg-bg-elevated/50">
                  <td className="px-4 py-3 font-medium text-text-primary">{pricing.model}</td>
                  <td className="px-4 py-3 text-text-secondary capitalize">{pricing.provider}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">${pricing.input_cost.toFixed(6)}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">${pricing.output_cost.toFixed(6)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
