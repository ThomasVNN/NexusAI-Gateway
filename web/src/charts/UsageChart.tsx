import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';

interface UsageDataPoint {
  time: string;
  requests: number;
  tokens: number;
}

interface UsageChartProps {
  data: UsageDataPoint[];
  timeRange?: '1h' | '24h' | '7d' | '30d';
  onTimeRangeChange?: (range: '1h' | '24h' | '7d' | '30d') => void;
}

const timeRanges = [
  { id: '1h' as const, label: '1H' },
  { id: '24h' as const, label: '24H' },
  { id: '7d' as const, label: '7D' },
  { id: '30d' as const, label: '30D' },
];

export function UsageChart({ data, timeRange = '24h', onTimeRangeChange }: UsageChartProps) {
  return (
    <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-text-primary">Request Volume</h3>
        {onTimeRangeChange && (
          <div className="flex gap-1 bg-bg-secondary rounded-lg p-1">
            {timeRanges.map((range) => (
              <button
                key={range.id}
                onClick={() => onTimeRangeChange(range.id)}
                className={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
                  timeRange === range.id
                    ? 'bg-accent-primary text-white'
                    : 'text-text-tertiary hover:text-text-secondary'
                }`}
              >
                {range.label}
              </button>
            ))}
          </div>
        )}
      </div>
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id="colorRequests" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#8B5CF6" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#8B5CF6" stopOpacity={0} />
              </linearGradient>
              <linearGradient id="colorTokens" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="#34D399" stopOpacity={0.3} />
                <stop offset="95%" stopColor="#34D399" stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="#27272A" vertical={false} />
            <XAxis
              dataKey="time"
              stroke="#71717A"
              fontSize={12}
              tickLine={false}
              axisLine={false}
            />
            <YAxis
              stroke="#71717A"
              fontSize={12}
              tickLine={false}
              axisLine={false}
              tickFormatter={(value) => value >= 1000 ? `${value / 1000}k` : value}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: '#1F1F23',
                border: '1px solid #27272A',
                borderRadius: '8px',
                fontSize: '12px',
              }}
              labelStyle={{ color: '#FAFAFA' }}
            />
            <Area
              type="monotone"
              dataKey="requests"
              stroke="#8B5CF6"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#colorRequests)"
              name="Requests"
            />
            <Area
              type="monotone"
              dataKey="tokens"
              stroke="#34D399"
              strokeWidth={2}
              fillOpacity={1}
              fill="url(#colorTokens)"
              name="Tokens (÷100)"
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
      <div className="flex items-center justify-center gap-6 mt-4">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-accent-primary" />
          <span className="text-xs text-text-tertiary">Requests</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-success" />
          <span className="text-xs text-text-tertiary">Tokens (÷100)</span>
        </div>
      </div>
    </div>
  );
}
