import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Cell,
} from 'recharts';

interface LatencyDataPoint {
  range: string;
  count: number;
  latency: number;
}

interface LatencyChartProps {
  data: LatencyDataPoint[];
  title?: string;
  p50?: number;
  p95?: number;
  p99?: number;
}

const getLatencyColor = (latency: number): string => {
  if (latency < 100) return '#34D399';
  if (latency < 300) return '#60A5FA';
  if (latency < 500) return '#FBBF24';
  return '#F87171';
};

export function LatencyChart({
  data,
  title = 'Latency Distribution',
  p50,
  p95,
  p99,
}: LatencyChartProps) {
  return (
    <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-medium text-text-primary">{title}</h3>
        {(p50 || p95 || p99) && (
          <div className="flex gap-3 text-xs">
            {p50 && (
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 rounded-full bg-success" />
                <span className="text-text-tertiary">P50: {p50}ms</span>
              </div>
            )}
            {p95 && (
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 rounded-full bg-warning" />
                <span className="text-text-tertiary">P95: {p95}ms</span>
              </div>
            )}
            {p99 && (
              <div className="flex items-center gap-1">
                <div className="w-2 h-2 rounded-full bg-error" />
                <span className="text-text-tertiary">P99: {p99}ms</span>
              </div>
            )}
          </div>
        )}
      </div>
      <div className="h-48">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 10, right: 10, left: 0, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#27272A" vertical={false} />
            <XAxis
              dataKey="range"
              stroke="#71717A"
              fontSize={11}
              tickLine={false}
              axisLine={false}
            />
            <YAxis
              stroke="#71717A"
              fontSize={11}
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
              formatter={(value: number, name: string, props: { payload?: LatencyDataPoint }) => [
                `${value} requests`,
                props.payload?.latency ? `${props.payload.latency}ms avg` : name,
              ]}
            />
            <Bar dataKey="count" radius={[4, 4, 0, 0]}>
              {data.map((entry, index) => (
                <Cell
                  key={`cell-${index}`}
                  fill={getLatencyColor(entry.latency)}
                  fillOpacity={0.8}
                />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
      <div className="flex items-center justify-center gap-6 mt-4">
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-success" />
          <span className="text-xs text-text-tertiary">&lt;100ms</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-info" />
          <span className="text-xs text-text-tertiary">100-300ms</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-warning" />
          <span className="text-xs text-text-tertiary">300-500ms</span>
        </div>
        <div className="flex items-center gap-2">
          <div className="w-3 h-3 rounded-full bg-error" />
          <span className="text-xs text-text-tertiary">&gt;500ms</span>
        </div>
      </div>
    </div>
  );
}
