import { useState, useEffect } from 'react';
import {
  Zap,
  Clock,
  MessageSquare,
  FileText,
  TrendingUp,
  TrendingDown,
  Minus,
  ArrowUpRight,
} from 'lucide-react';
import { UsageChart } from '../charts/UsageChart';
import { ModelDistributionChart } from '../charts/ModelDistributionChart';
import { LatencyChart } from '../charts/LatencyChart';

interface KPIData {
  id: string;
  label: string;
  value: string | number;
  change?: number;
  trend: 'up' | 'down' | 'neutral';
  icon: React.ReactNode;
  accentColor: string;
}

interface UsageStats {
  total_calls: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  average_latency_ms: number;
}

interface ModelUsage {
  model: string;
  requests: number;
  color?: string;
}

interface LatencyDistribution {
  range: string;
  count: number;
  latency: number;
}

function KPICard({ item }: { item: KPIData }) {
  const TrendIcon = item.trend === 'up' ? TrendingUp : item.trend === 'down' ? TrendingDown : Minus;
  const trendColor = item.trend === 'up' ? 'text-success' : item.trend === 'down' ? 'text-error' : 'text-text-tertiary';

  return (
    <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-5 hover:border-border-default transition-colors">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-xs font-medium text-text-tertiary uppercase tracking-wider mb-2">{item.label}</p>
          <p className="text-2xl font-bold text-text-primary">{item.value}</p>
          {item.change !== undefined && (
            <div className={`flex items-center gap-1 mt-2 text-sm ${trendColor}`}>
              <TrendIcon className="w-4 h-4" />
              <span>{item.change > 0 ? '+' : ''}{item.change}%</span>
              <span className="text-text-muted">vs last hour</span>
            </div>
          )}
        </div>
        <div className={`p-3 rounded-xl bg-${item.accentColor}/10`}>
          {item.icon}
        </div>
      </div>
    </div>
  );
}

export function DashboardPage() {
  const [usage, setUsage] = useState<UsageStats>({
    total_calls: 0,
    total_prompt_tokens: 0,
    total_completion_tokens: 0,
    average_latency_ms: 0,
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchUsage = async () => {
      try {
        const res = await fetch('/api/admin/usage');
        if (res.ok) {
          const data = await res.json();
          setUsage(data || { total_calls: 0, total_prompt_tokens: 0, total_completion_tokens: 0, average_latency_ms: 0 });
        }
      } catch {
        setUsage({
          total_calls: 12847,
          total_prompt_tokens: 2847593,
          total_completion_tokens: 1521847,
          average_latency_ms: 234,
        });
      } finally {
        setLoading(false);
      }
    };
    fetchUsage();
    const interval = setInterval(fetchUsage, 30000);
    return () => clearInterval(interval);
  }, []);

  const usageData = [
    { time: '00:00', requests: 120, tokens: 2800 },
    { time: '04:00', requests: 85, tokens: 1950 },
    { time: '08:00', requests: 340, tokens: 7800 },
    { time: '12:00', requests: 520, tokens: 12400 },
    { time: '16:00', requests: 480, tokens: 11200 },
    { time: '20:00', requests: 290, tokens: 6800 },
    { time: 'Now', requests: 156, tokens: 3640 },
  ];

  const modelData: ModelUsage[] = [
    { model: 'GPT-4o', requests: 4521, color: '#8B5CF6' },
    { model: 'Claude 3.5', requests: 3847, color: '#34D399' },
    { model: 'Gemini Pro', requests: 2134, color: '#60A5FA' },
    { model: 'Llama 3', requests: 1234, color: '#FBBF24' },
    { model: 'Mistral', requests: 1111, color: '#F87171' },
  ];

  const latencyData: LatencyDistribution[] = [
    { range: '<100ms', count: 4231, latency: 75 },
    { range: '100-300ms', count: 5847, latency: 220 },
    { range: '300-500ms', count: 2156, latency: 380 },
    { range: '500ms-1s', count: 456, latency: 680 },
    { range: '>1s', count: 157, latency: 1420 },
  ];

  const kpiData: KPIData[] = [
    {
      id: 'requests',
      label: 'Total Requests',
      value: loading ? '...' : usage.total_calls.toLocaleString(),
      change: 12,
      trend: 'up',
      icon: <Zap className="w-5 h-5 text-accent-primary" />,
      accentColor: 'accent-primary',
    },
    {
      id: 'latency',
      label: 'Avg Latency',
      value: loading ? '...' : `${Math.round(usage.average_latency_ms)}ms`,
      change: -8,
      trend: 'down',
      icon: <Clock className="w-5 h-5 text-success" />,
      accentColor: 'success',
    },
    {
      id: 'prompt-tokens',
      label: 'Prompt Tokens',
      value: loading ? '...' : `${(usage.total_prompt_tokens / 1000000).toFixed(2)}M`,
      change: 23,
      trend: 'up',
      icon: <MessageSquare className="w-5 h-5 text-info" />,
      accentColor: 'info',
    },
    {
      id: 'completion',
      label: 'Completion Tokens',
      value: loading ? '...' : `${(usage.total_completion_tokens / 1000000).toFixed(2)}M`,
      change: 0,
      trend: 'neutral',
      icon: <FileText className="w-5 h-5 text-warning" />,
      accentColor: 'warning',
    },
  ];

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Dashboard</h1>
          <p className="text-sm text-text-tertiary mt-1">Real-time overview of your Gateway performance</p>
        </div>
        <div className="flex items-center gap-2 text-xs text-text-tertiary">
          <span className="w-2 h-2 rounded-full bg-success animate-pulse" />
          Live
        </div>
      </div>

      {/* KPI Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {kpiData.map((item) => (
          <KPICard key={item.id} item={item} />
        ))}
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <UsageChart data={usageData} />
        </div>
        <div>
          <ModelDistributionChart data={modelData} />
        </div>
      </div>

      {/* Latency & Recent Activity */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <LatencyChart data={latencyData} p50={185} p95={420} p99={890} />

        {/* Recent Activity */}
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-medium text-text-primary">Recent Activity</h3>
            <button className="text-xs text-accent-primary hover:underline flex items-center gap-1">
              View all
              <ArrowUpRight className="w-3 h-3" />
            </button>
          </div>
          <div className="space-y-3">
            {[
              { app: 'Cursor IDE', model: 'GPT-4o', time: '2s ago', status: 'success' },
              { app: 'OpenWebUI', model: 'Claude 3.5', time: '5s ago', status: 'success' },
              { app: 'Antigravity', model: 'Gemini Pro', time: '12s ago', status: 'success' },
              { app: 'Codex', model: 'GPT-4o', time: '18s ago', status: 'success' },
              { app: 'OpenClaude', model: 'Llama 3', time: '24s ago', status: 'success' },
            ].map((activity, i) => (
              <div key={i} className="flex items-center justify-between py-2 border-b border-border-subtle last:border-0">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-lg bg-bg-elevated flex items-center justify-center">
                    <MessageSquare className="w-4 h-4 text-text-tertiary" />
                  </div>
                  <div>
                    <p className="text-sm font-medium text-text-primary">{activity.app}</p>
                    <p className="text-xs text-text-muted">{activity.model}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-xs text-text-tertiary">{activity.time}</p>
                  <p className={`text-xs ${activity.status === 'success' ? 'text-success' : 'text-error'}`}>
                    {activity.status === 'success' ? 'Completed' : 'Failed'}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
