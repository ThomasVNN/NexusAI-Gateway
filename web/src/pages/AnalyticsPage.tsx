import { useState } from 'react';
import {
  BarChart3,
  Filter,
  Download,
  Calendar,
  TrendingUp,
} from 'lucide-react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  BarChart,
  Bar,
  LineChart,
  Line,
  Legend,
} from 'recharts';

type TimeRange = '1h' | '24h' | '7d' | '30d' | '90d';

interface RequestData {
  time: string;
  requests: number;
  latency: number;
  errors: number;
}

interface ModelData {
  model: string;
  requests: number;
  tokens: number;
  cost: number;
  avgLatency: number;
  color: string;
}

const timeRanges = [
  { id: '1h' as const, label: '1 Hour' },
  { id: '24h' as const, label: '24 Hours' },
  { id: '7d' as const, label: '7 Days' },
  { id: '30d' as const, label: '30 Days' },
  { id: '90d' as const, label: '90 Days' },
];

export function AnalyticsPage() {
  const [timeRange, setTimeRange] = useState<TimeRange>('7d');
  const [selectedModel, setSelectedModel] = useState<string>('all');

  const requestData: RequestData[] = [
    { time: 'Mon', requests: 2450, latency: 220, errors: 12 },
    { time: 'Tue', requests: 3120, latency: 245, errors: 8 },
    { time: 'Wed', requests: 2890, latency: 198, errors: 15 },
    { time: 'Thu', requests: 3650, latency: 267, errors: 22 },
    { time: 'Fri', requests: 4200, latency: 312, errors: 18 },
    { time: 'Sat', requests: 1890, latency: 185, errors: 5 },
    { time: 'Sun', requests: 1567, latency: 176, errors: 3 },
  ];

  const modelData: ModelData[] = [
    { model: 'GPT-4o', requests: 8942, tokens: 1847593, cost: 28.45, avgLatency: 285, color: '#8B5CF6' },
    { model: 'Claude 3.5', requests: 7621, tokens: 1623847, cost: 19.23, avgLatency: 245, color: '#34D399' },
    { model: 'Gemini Pro', requests: 4234, tokens: 892134, cost: 4.56, avgLatency: 198, color: '#60A5FA' },
    { model: 'Llama 3', requests: 3456, tokens: 723894, cost: 1.89, avgLatency: 156, color: '#FBBF24' },
    { model: 'Mistral', requests: 2134, tokens: 456123, cost: 1.12, avgLatency: 142, color: '#F87171' },
  ];

  const latencyData = [
    { time: '00:00', p50: 145, p95: 380, p99: 620 },
    { time: '04:00', p50: 132, p95: 345, p99: 580 },
    { time: '08:00', p50: 198, p95: 456, p99: 780 },
    { time: '12:00', p50: 245, p95: 520, p99: 890 },
    { time: '16:00', p50: 267, p95: 545, p99: 920 },
    { time: '20:00', p50: 234, p95: 498, p99: 845 },
    { time: 'Now', p50: 198, p95: 445, p99: 780 },
  ];

  const totalRequests = modelData.reduce((s, m) => s + m.requests, 0);
  const totalTokens = modelData.reduce((s, m) => s + m.tokens, 0);
  const totalCost = modelData.reduce((s, m) => s + m.cost, 0);
  const avgLatency = modelData.reduce((s, m) => s + m.avgLatency, 0) / modelData.length;

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary">Analytics</h1>
          <p className="text-sm text-text-tertiary mt-1">Detailed usage analytics and performance metrics</p>
        </div>
        <div className="flex items-center gap-3">
          <select
            value={selectedModel}
            onChange={(e) => setSelectedModel(e.target.value)}
            className="px-3 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-primary text-sm focus:border-border-focus focus:outline-none"
          >
            <option value="all">All Models</option>
            {modelData.map((m) => (
              <option key={m.model} value={m.model}>{m.model}</option>
            ))}
          </select>
          <div className="flex gap-1 bg-bg-tertiary rounded-lg p-1">
            {timeRanges.map((range) => (
              <button
                key={range.id}
                onClick={() => setTimeRange(range.id)}
                className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                  timeRange === range.id
                    ? 'bg-accent-primary text-white'
                    : 'text-text-tertiary hover:text-text-secondary'
                }`}
              >
                {range.label}
              </button>
            ))}
          </div>
          <button className="flex items-center gap-2 px-4 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-text-secondary hover:text-text-primary transition-colors">
            <Download className="w-4 h-4" />
            Export
          </button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-4 gap-4">
        {[
          { label: 'Total Requests', value: totalRequests.toLocaleString(), icon: BarChart3, color: 'text-accent-primary' },
          { label: 'Total Tokens', value: `${(totalTokens / 1000000).toFixed(2)}M`, icon: TrendingUp, color: 'text-success' },
          { label: 'Total Cost', value: `$${totalCost.toFixed(2)}`, icon: BarChart3, color: 'text-warning' },
          { label: 'Avg Latency', value: `${Math.round(avgLatency)}ms`, icon: BarChart3, color: 'text-info' },
        ].map((stat, i) => (
          <div key={i} className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-xs text-text-tertiary">{stat.label}</p>
                <p className="text-2xl font-bold text-text-primary mt-1">{stat.value}</p>
              </div>
              <stat.icon className={`w-8 h-8 ${stat.color}`} />
            </div>
          </div>
        ))}
      </div>

      {/* Charts Row 1 */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Request Volume */}
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <h3 className="text-sm font-medium text-text-primary mb-4">Request Volume</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={requestData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                <defs>
                  <linearGradient id="colorReq" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#8B5CF6" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#8B5CF6" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#27272A" vertical={false} />
                <XAxis dataKey="time" stroke="#71717A" fontSize={12} tickLine={false} axisLine={false} />
                <YAxis stroke="#71717A" fontSize={12} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={{ backgroundColor: '#1F1F23', border: '1px solid #27272A', borderRadius: '8px', fontSize: '12px' }} />
                <Area type="monotone" dataKey="requests" stroke="#8B5CF6" strokeWidth={2} fill="url(#colorReq)" name="Requests" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Error Rate */}
        <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
          <h3 className="text-sm font-medium text-text-primary mb-4">Error Rate</h3>
          <div className="h-64">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={requestData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#27272A" vertical={false} />
                <XAxis dataKey="time" stroke="#71717A" fontSize={12} tickLine={false} axisLine={false} />
                <YAxis stroke="#71717A" fontSize={12} tickLine={false} axisLine={false} />
                <Tooltip contentStyle={{ backgroundColor: '#1F1F23', border: '1px solid #27272A', borderRadius: '8px', fontSize: '12px' }} />
                <Line type="monotone" dataKey="errors" stroke="#F87171" strokeWidth={2} name="Errors" dot={{ fill: '#F87171' }} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      {/* Latency Percentiles */}
      <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
        <h3 className="text-sm font-medium text-text-primary mb-4">Latency Percentiles (ms)</h3>
        <div className="h-64">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={latencyData} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#27272A" vertical={false} />
              <XAxis dataKey="time" stroke="#71717A" fontSize={12} tickLine={false} axisLine={false} />
              <YAxis stroke="#71717A" fontSize={12} tickLine={false} axisLine={false} />
              <Tooltip contentStyle={{ backgroundColor: '#1F1F23', border: '1px solid #27272A', borderRadius: '8px', fontSize: '12px' }} />
              <Legend />
              <Line type="monotone" dataKey="p50" stroke="#34D399" strokeWidth={2} name="P50" />
              <Line type="monotone" dataKey="p95" stroke="#FBBF24" strokeWidth={2} name="P95" />
              <Line type="monotone" dataKey="p99" stroke="#F87171" strokeWidth={2} name="P99" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Model Comparison */}
      <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
        <h3 className="text-sm font-medium text-text-primary mb-4">Model Comparison</h3>
        <div className="h-80">
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={modelData} layout="vertical" margin={{ top: 10, right: 30, left: 100, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#27272A" horizontal={false} />
              <XAxis type="number" stroke="#71717A" fontSize={12} tickLine={false} axisLine={false} />
              <YAxis type="category" dataKey="model" stroke="#71717A" fontSize={12} tickLine={false} axisLine={false} />
              <Tooltip contentStyle={{ backgroundColor: '#1F1F23', border: '1px solid #27272A', borderRadius: '8px', fontSize: '12px' }} />
              <Legend />
              <Bar dataKey="requests" fill="#8B5CF6" name="Requests" radius={[0, 4, 4, 0]} />
              <Bar dataKey="tokens" fill="#34D399" name="Tokens (÷100)" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Model Details Table */}
      <div className="bg-bg-tertiary border border-border-subtle rounded-xl overflow-hidden">
        <div className="p-4 border-b border-border-subtle">
          <h3 className="text-sm font-medium text-text-primary">Model Details</h3>
        </div>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead className="bg-bg-secondary">
              <tr>
                <th className="px-4 py-3 text-left text-xs font-medium text-text-tertiary uppercase">Model</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Requests</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Tokens</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Cost</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">Avg Latency</th>
                <th className="px-4 py-3 text-right text-xs font-medium text-text-tertiary uppercase">% of Total</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border-subtle">
              {modelData.map((model, i) => (
                <tr key={i} className="hover:bg-bg-elevated/50">
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-2">
                      <div className="w-3 h-3 rounded-full" style={{ backgroundColor: model.color }} />
                      <span className="font-medium text-text-primary">{model.model}</span>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right text-text-secondary">{model.requests.toLocaleString()}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">{model.tokens.toLocaleString()}</td>
                  <td className="px-4 py-3 text-right font-medium text-accent-primary">${model.cost.toFixed(2)}</td>
                  <td className="px-4 py-3 text-right text-text-secondary">{model.avgLatency}ms</td>
                  <td className="px-4 py-3 text-right text-text-secondary">
                    {((model.requests / totalRequests) * 100).toFixed(1)}%
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
