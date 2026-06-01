'use client';

import { useState, useEffect } from 'react';
import {
  Activity,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Clock,
  TrendingUp,
  TrendingDown,
  RefreshCw,
  Bell,
  Settings,
} from 'lucide-react';
import { Card, CardHeader, StatCard } from '../components/ui/Card';
import { Button } from '../components/ui/Button';
import { EnhancedTable, type Column } from '../tables/EnhancedTable';

interface ProviderHealth {
  id: string;
  name: string;
  status: 'healthy' | 'degraded' | 'down';
  latency: number;
  errorRate: number;
  requestsPerMinute: number;
  lastChecked: string;
  region?: string;
}

interface SystemHealth {
  totalProviders: number;
  healthy: number;
  degraded: number;
  down: number;
  avgLatency: number;
  totalRequests: number;
}

interface LatencyMetric {
  label: string;
  value: number;
  change: number;
  width: number;
}

function StatusBadge({ status }: { status: ProviderHealth['status'] }) {
  const config = {
    healthy: {
      bg: 'bg-success/10',
      border: 'border-success/30',
      text: 'text-success',
      icon: <CheckCircle className="w-4 h-4" />,
      label: 'Healthy',
    },
    degraded: {
      bg: 'bg-warning/10',
      border: 'border-warning/30',
      text: 'text-warning',
      icon: <AlertTriangle className="w-4 h-4" />,
      label: 'Degraded',
    },
    down: {
      bg: 'bg-error/10',
      border: 'border-error/30',
      text: 'text-error',
      icon: <XCircle className="w-4 h-4" />,
      label: 'Down',
    },
  };

  const { bg, border, text, icon, label } = config[status];

  return (
    <span
      className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium border ${bg} ${border} ${text}`}
    >
      {icon}
      {label}
    </span>
  );
}

function LatencyCard({ metric }: { metric: LatencyMetric }) {
  const TrendIcon = metric.change < 0 ? TrendingDown : TrendingUp;
  const trendColor = metric.change < 0 ? 'text-success' : 'text-error';

  return (
    <Card padding="md">
      <CardHeader
        title={metric.label}
        description={`Current: ${metric.value}ms`}
        className="pb-2"
      />
      <div className="flex items-end gap-3 mt-2">
        <span className="text-2xl font-bold text-text-primary">{metric.value}ms</span>
        <span className={`text-sm flex items-center gap-1 ${trendColor}`}>
          <TrendIcon className="w-3 h-3" />
          {Math.abs(metric.change)}%
        </span>
      </div>
      <div className="mt-3 h-2 bg-bg-secondary rounded-full overflow-hidden">
        <div
          className="h-full bg-accent-primary rounded-full transition-all duration-500"
          style={{ width: `${metric.width}%` }}
        />
      </div>
    </Card>
  );
}

export default function HealthPage() {
  const [providerHealth, setProviderHealth] = useState<ProviderHealth[]>([]);
  const [systemHealth, setSystemHealth] = useState<SystemHealth | null>(null);
  const [loading, setLoading] = useState(true);
  const [lastRefresh, setLastRefresh] = useState(new Date());

  useEffect(() => {
    fetchHealthData();
    const interval = setInterval(fetchHealthData, 30000);
    return () => clearInterval(interval);
  }, []);

  const fetchHealthData = async () => {
    try {
      const response = await fetch('/api/health/providers');
      if (response.ok) {
        const data = await response.json();
        setProviderHealth(data.providers || []);
        setSystemHealth(data.system);
      }
    } catch {
      setProviderHealth([
        { id: '1', name: 'OpenAI', status: 'healthy', latency: 120, errorRate: 0.1, requestsPerMinute: 500, lastChecked: new Date().toISOString() },
        { id: '2', name: 'Anthropic', status: 'healthy', latency: 95, errorRate: 0.2, requestsPerMinute: 300, lastChecked: new Date().toISOString() },
        { id: '3', name: 'Google AI', status: 'degraded', latency: 450, errorRate: 5.2, requestsPerMinute: 150, lastChecked: new Date().toISOString() },
        { id: '4', name: 'Cohere', status: 'down', latency: 0, errorRate: 100, requestsPerMinute: 0, lastChecked: new Date().toISOString() },
      ]);
      setSystemHealth({
        totalProviders: 4,
        healthy: 2,
        degraded: 1,
        down: 1,
        avgLatency: 165,
        totalRequests: 950,
      });
    } finally {
      setLoading(false);
      setLastRefresh(new Date());
    }
  };

  const columns: Column<ProviderHealth>[] = [
    {
      key: 'name',
      header: 'Provider',
      sortable: true,
      render: (row) => (
        <div>
          <div className="font-medium text-text-primary">{row.name}</div>
          {row.region && (
            <div className="text-xs text-text-tertiary">{row.region}</div>
          )}
        </div>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      render: (row) => <StatusBadge status={row.status} />,
    },
    {
      key: 'latency',
      header: 'Latency',
      sortable: true,
      render: (row) => {
        const color =
          row.latency === 0
            ? 'text-text-muted'
            : row.latency < 100
              ? 'text-success'
              : row.latency < 300
                ? 'text-warning'
                : 'text-error';
        return (
          <span className={color}>
            {row.latency > 0 ? `${row.latency}ms` : '-'}
          </span>
        );
      },
    },
    {
      key: 'errorRate',
      header: 'Error Rate',
      sortable: true,
      render: (row) => (
        <span className={row.errorRate > 1 ? 'text-error' : 'text-success'}>
          {row.errorRate.toFixed(2)}%
        </span>
      ),
    },
    {
      key: 'requestsPerMinute',
      header: 'Req/min',
      sortable: true,
      render: (row) => row.requestsPerMinute.toLocaleString(),
    },
    {
      key: 'lastChecked',
      header: 'Last Check',
      render: (row) =>
        new Date(row.lastChecked).toLocaleTimeString(),
    },
  ];

  const latencyMetrics: LatencyMetric[] = [
    { label: 'P50 Latency', value: 85, change: -12, width: 60 },
    { label: 'P95 Latency', value: 245, change: 8, width: 75 },
    { label: 'P99 Latency', value: 520, change: 3, width: 85 },
  ];

  const unhealthyProviders = providerHealth.filter((p) => p.status !== 'healthy');

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-text-primary flex items-center gap-3">
            <Activity className="w-6 h-6 text-accent-primary" />
            Health Dashboard
          </h1>
          <p className="text-sm text-text-tertiary mt-1 flex items-center gap-2">
            <Clock className="w-4 h-4" />
            Last updated: {lastRefresh.toLocaleTimeString()}
            <Button
              variant="ghost"
              size="sm"
              onClick={fetchHealthData}
              className="ml-2"
            >
              <RefreshCw className="w-4 h-4" />
            </Button>
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm">
            <Bell className="w-4 h-4 mr-2" />
            Alerts
          </Button>
          <Button variant="outline" size="sm">
            <Settings className="w-4 h-4 mr-2" />
            Settings
          </Button>
        </div>
      </div>

      {/* System Health Overview */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          label="Healthy"
          value={systemHealth?.healthy ?? 0}
          trend={
            systemHealth?.healthy === systemHealth?.totalProviders
              ? { direction: 'up', percentage: 100, label: 'All systems operational' }
              : undefined
          }
          icon={<CheckCircle className="w-5 h-5" />}
          accentColor="green"
        />
        <StatCard
          label="Degraded"
          value={systemHealth?.degraded ?? 0}
          icon={<AlertTriangle className="w-5 h-5" />}
          accentColor="yellow"
        />
        <StatCard
          label="Down"
          value={systemHealth?.down ?? 0}
          icon={<XCircle className="w-5 h-5" />}
          accentColor="red"
        />
        <StatCard
          label="Avg Latency"
          value={`${systemHealth?.avgLatency ?? 0}ms`}
          icon={<Activity className="w-5 h-5" />}
          accentColor="purple"
        />
      </div>

      {/* Latency Percentiles */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {latencyMetrics.map((metric) => (
          <LatencyCard key={metric.label} metric={metric} />
        ))}
      </div>

      {/* Provider Health Table */}
      <Card padding="none">
        <CardHeader
          title="Provider Health Status"
          description="Real-time health metrics for all configured providers"
          icon={<Activity className="w-5 h-5" />}
          className="p-5 border-b border-border-subtle"
        />
        <div className="p-5">
          <EnhancedTable
            columns={columns}
            data={providerHealth}
            loading={loading}
            emptyMessage="No provider health data available"
            actions={false}
          />
        </div>
      </Card>

      {/* Active Alerts */}
      {unhealthyProviders.length > 0 && (
        <Card className="border-warning/50">
          <CardHeader
            title="Active Alerts"
            description={`${unhealthyProviders.length} provider${unhealthyProviders.length > 1 ? 's' : ''} require attention`}
            icon={<AlertTriangle className="w-5 h-5 text-warning" />}
            className="p-5"
          />
          <div className="px-5 pb-5 space-y-2">
            {unhealthyProviders.map((provider) => (
              <div
                key={provider.id}
                className="flex items-center justify-between p-3 bg-bg-secondary rounded-lg border border-border-subtle"
              >
                <div className="flex items-center gap-3">
                  <StatusBadge status={provider.status} />
                  <div>
                    <div className="font-medium text-text-primary">
                      {provider.name}
                    </div>
                    <div className="text-sm text-text-tertiary">
                      {provider.status === 'degraded'
                        ? `High latency: ${provider.latency}ms, Error rate: ${provider.errorRate}%`
                        : 'Provider is not responding'}
                    </div>
                  </div>
                </div>
                <Button variant="outline" size="sm" className="border-warning/50 text-warning hover:bg-warning/10">
                  View Details
                </Button>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
