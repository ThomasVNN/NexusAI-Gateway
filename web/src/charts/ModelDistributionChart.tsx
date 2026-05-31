import {
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Legend,
  Tooltip,
} from 'recharts';

interface ModelData {
  model: string;
  requests: number;
  color: string;
}

interface ModelDistributionChartProps {
  data: ModelData[];
  title?: string;
}

const COLORS = [
  '#8B5CF6',
  '#34D399',
  '#60A5FA',
  '#FBBF24',
  '#F87171',
  '#A78BFA',
  '#10B981',
  '#EC4899',
];

export function ModelDistributionChart({ data, title = 'Model Distribution' }: ModelDistributionChartProps) {
  const chartData = data.map((item, index) => ({
    ...item,
    color: item.color || COLORS[index % COLORS.length],
  }));

  const totalRequests = data.reduce((sum, item) => sum + item.requests, 0);

  return (
    <div className="bg-bg-tertiary border border-border-subtle rounded-xl p-4">
      <h3 className="text-sm font-medium text-text-primary mb-4">{title}</h3>
      <div className="h-64">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={chartData}
              cx="50%"
              cy="50%"
              innerRadius={60}
              outerRadius={90}
              paddingAngle={2}
              dataKey="requests"
              nameKey="model"
            >
              {chartData.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={entry.color} stroke="transparent" />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: '#1F1F23',
                border: '1px solid #27272A',
                borderRadius: '8px',
                fontSize: '12px',
              }}
              formatter={(value: number, name: string) => [
                `${((value / totalRequests) * 100).toFixed(1)}%`,
                name,
              ]}
            />
            <Legend
              layout="vertical"
              align="right"
              verticalAlign="middle"
              iconType="circle"
              iconSize={8}
              formatter={(value) => (
                <span className="text-xs text-text-secondary">{value}</span>
              )}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>
      <div className="mt-4 pt-4 border-t border-border-subtle">
        <div className="flex items-center justify-between text-sm">
          <span className="text-text-tertiary">Total Requests</span>
          <span className="font-medium text-text-primary">
            {totalRequests.toLocaleString()}
          </span>
        </div>
      </div>
    </div>
  );
}
