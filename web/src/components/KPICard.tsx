import React from 'react';

export interface KPIData {
  id: string;
  label: string;
  value: string | number;
  trend?: {
    direction: 'up' | 'down' | 'neutral';
    percentage: number;
    label?: string;
  };
  icon?: React.ReactNode;
  accentColor?: 'purple' | 'green' | 'blue' | 'red' | 'yellow';
}

interface KPICardProps {
  data: KPIData;
  className?: string;
}

interface KPIGridProps {
  items: KPIData[];
  columns?: 2 | 3 | 4;
  className?: string;
}

const accentStyles = {
  purple: { bg: 'bg-[#8B5CF6]/10', border: 'border-[#8B5CF6]/30', icon: 'text-accent-primary' },
  green: { bg: 'bg-[#34D399]/10', border: 'border-[#34D399]/30', icon: 'text-[#34D399]' },
  blue: { bg: 'bg-[#60A5FA]/10', border: 'border-[#60A5FA]/30', icon: 'text-[#60A5FA]' },
  red: { bg: 'bg-[#F87171]/10', border: 'border-[#F87171]/30', icon: 'text-[#F87171]' },
  yellow: { bg: 'bg-[#FBBF24]/10', border: 'border-[#FBBF24]/30', icon: 'text-[#FBBF24]' },
};

function TrendIcon({ direction }: { direction: 'up' | 'down' | 'neutral' }) {
  if (direction === 'up') {
    return (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M5 15l7-7 7 7" />
      </svg>
    );
  }
  if (direction === 'down') {
    return (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
      </svg>
    );
  }
  return (
    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M20 12H4" />
    </svg>
  );
}

export function KPICard({ data, className = '' }: KPICardProps) {
  const style = accentStyles[data.accentColor || 'purple'];

  return (
    <div
      className={`relative overflow-hidden ${style.bg} ${style.border} rounded-xl p-5 transition-all duration-200 hover:shadow-lg ${className}`}
      style={{ animation: 'fadeIn 200ms ease-out forwards' }}
    >
      {/* Decorative Glow */}
      <div className="absolute -top-10 -right-10 w-32 h-32 bg-accent-primary/10 rounded-full blur-2xl pointer-events-none" />

      <div className="relative z-10">
        {/* Header Row */}
        <div className="flex items-start justify-between mb-3">
          <span className="text-xs font-medium text-text-tertiary uppercase tracking-wider">
            {data.label}
          </span>
          {data.icon && (
            <div className={`${style.icon} opacity-60`}>
              {data.icon}
            </div>
          )}
        </div>

        {/* Value */}
        <div className="text-3xl font-bold text-text-primary tracking-tight mb-2">
          {typeof data.value === 'number' ? data.value.toLocaleString() : data.value}
        </div>

        {/* Trend */}
        {data.trend && (
          <div className="flex items-center gap-2">
            <span
              className={`flex items-center gap-1 text-sm font-medium ${
                data.trend.direction === 'up' ? 'text-[#34D399]' : ''
              } ${data.trend.direction === 'down' ? 'text-[#F87171]' : ''} ${
                data.trend.direction === 'neutral' ? 'text-text-tertiary' : ''
              }`}
            >
              <TrendIcon direction={data.trend.direction} />
              {data.trend.percentage}%
            </span>
            {data.trend.label && (
              <span className="text-xs text-text-tertiary">
                {data.trend.label}
              </span>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

export function KPIGrid({ items, columns = 4, className = '' }: KPIGridProps) {
  const gridCols = {
    2: 'grid-cols-1 sm:grid-cols-2',
    3: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4',
  };

  return (
    <div className={`grid ${gridCols[columns]} gap-4 ${className}`}>
      {items.map((item, index) => (
        <div
          key={item.id}
          style={{ animationDelay: `${index * 50}ms`, animationFillMode: 'both' }}
        >
          <KPICard data={item} />
        </div>
      ))}
    </div>
  );
}

export default KPICard;
