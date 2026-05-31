import React from 'react';

// ============================================================================
// Type Definitions
// ============================================================================

export type CardVariant = 'default' | 'elevated' | 'interactive' | 'bordered';
export type CardPadding = 'none' | 'sm' | 'md' | 'lg';

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: CardVariant;
  padding?: CardPadding;
  hoverable?: boolean;
  accentBorder?: boolean;
}

export interface StatCardProps extends Omit<CardProps, 'variant'> {
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

export interface MetricCardProps extends Omit<CardProps, 'variant'> {
  title: string;
  value: string | number;
  unit?: string;
  change?: {
    value: number;
    direction: 'up' | 'down';
    timeframe?: string;
  };
  sparklineData?: number[];
}

// ============================================================================
// Variant Styles
// ============================================================================

const variantStyles: Record<CardVariant, string> = {
  default: `
    bg-bg-tertiary border border-border-subtle
  `,
  elevated: `
    bg-bg-secondary border border-border-default shadow-md
  `,
  interactive: `
    bg-bg-tertiary border border-border-subtle
    cursor-pointer
    transition-all duration-200
    hover:border-border-default hover:shadow-lg hover:-translate-y-0.5
    active:translate-y-0
  `,
  bordered: `
    bg-transparent border border-border-default
  `,
};

const paddingStyles: Record<CardPadding, string> = {
  none: '',
  sm: 'p-3',
  md: 'p-5',
  lg: 'p-6',
};

// ============================================================================
// Card Component
// ============================================================================

export function Card({
  variant = 'default',
  padding = 'md',
  hoverable = false,
  accentBorder = false,
  className = '',
  children,
  ...props
}: CardProps) {
  const isInteractive = hoverable || variant === 'interactive';

  return (
    <div
      className={`
        rounded-xl
        ${variantStyles[variant]}
        ${paddingStyles[padding]}
        ${isInteractive ? 'cursor-pointer transition-all duration-200 hover:border-border-default hover:shadow-lg hover:-translate-y-0.5' : ''}
        ${accentBorder ? 'border-l-4 border-l-accent-primary' : ''}
        ${className}
      `.trim().replace(/\s+/g, ' ')}
      role={isInteractive ? 'button' : undefined}
      tabIndex={isInteractive ? 0 : undefined}
      {...props}
    >
      {children}
    </div>
  );
}

// ============================================================================
// Card Header
// ============================================================================

export interface CardHeaderProps extends React.HTMLAttributes<HTMLDivElement> {
  title: string;
  description?: string;
  action?: React.ReactNode;
  icon?: React.ReactNode;
}

export function CardHeader({
  title,
  description,
  action,
  icon,
  className = '',
  ...props
}: CardHeaderProps) {
  return (
    <div
      className={`flex items-start justify-between gap-4 ${className}`}
      {...props}
    >
      <div className="flex items-start gap-3 flex-1 min-w-0">
        {icon && (
          <div className="flex-shrink-0 w-10 h-10 rounded-lg bg-accent-primary/10 flex items-center justify-center text-accent-primary">
            {icon}
          </div>
        )}
        <div className="min-w-0">
          <h3 className="text-base font-semibold text-text-primary truncate">
            {title}
          </h3>
          {description && (
            <p className="text-sm text-text-secondary mt-0.5 line-clamp-2">
              {description}
            </p>
          )}
        </div>
      </div>
      {action && (
        <div className="flex-shrink-0">
          {action}
        </div>
      )}
    </div>
  );
}

// ============================================================================
// Stat Card Component
// ============================================================================

const accentColorStyles = {
  purple: {
    bg: 'bg-accent-primary/10',
    border: 'border-accent-primary/30',
    text: 'text-accent-primary',
  },
  green: {
    bg: 'bg-success/10',
    border: 'border-success/30',
    text: 'text-success',
  },
  blue: {
    bg: 'bg-info/10',
    border: 'border-info/30',
    text: 'text-info',
  },
  red: {
    bg: 'bg-error/10',
    border: 'border-error/30',
    text: 'text-error',
  },
  yellow: {
    bg: 'bg-warning/10',
    border: 'border-warning/30',
    text: 'text-warning',
  },
};

function TrendIndicator({ trend }: { trend: StatCardProps['trend'] }) {
  if (!trend) return null;

  const { direction, percentage, label } = trend;
  const trendColor = direction === 'up' ? 'text-success' : direction === 'down' ? 'text-error' : 'text-text-tertiary';
  const TrendIcon = () => {
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
  };

  return (
    <div className={`flex items-center gap-1.5 ${trendColor}`}>
      <TrendIcon />
      <span className="text-sm font-medium">
        {direction === 'neutral' ? '' : `${direction === 'up' ? '+' : '-'}`}{percentage}%
      </span>
      {label && (
        <span className="text-xs text-text-tertiary">{label}</span>
      )}
    </div>
  );
}

export function StatCard({
  label,
  value,
  trend,
  icon,
  accentColor = 'purple',
  className = '',
  ...props
}: StatCardProps) {
  const colorStyle = accentColorStyles[accentColor];

  return (
    <div
      className={`
        relative overflow-hidden
        ${colorStyle.bg} ${colorStyle.border}
        rounded-xl p-5
        border
        transition-all duration-200
        hover:shadow-lg
        ${className}
      `.trim().replace(/\s+/g, ' ')}
      {...props}
    >
      {/* Decorative Glow */}
      <div className="absolute -top-10 -right-10 w-32 h-32 bg-accent-primary/10 rounded-full blur-2xl pointer-events-none" />

      <div className="relative z-10">
        {/* Header Row */}
        <div className="flex items-start justify-between mb-3">
          <span className="text-xs font-medium text-text-tertiary uppercase tracking-wider">
            {label}
          </span>
          {icon && (
            <div className={`${colorStyle.text} opacity-60`}>
              {icon}
            </div>
          )}
        </div>

        {/* Value */}
        <div className="text-3xl font-bold text-text-primary tracking-tight mb-2">
          {typeof value === 'number' ? value.toLocaleString() : value}
        </div>

        {/* Trend */}
        {trend && <TrendIndicator trend={trend} />}
      </div>
    </div>
  );
}

// ============================================================================
// Metric Card Component
// ============================================================================

export function MetricCard({
  title,
  value,
  unit,
  change,
  className = '',
  ...props
}: MetricCardProps) {
  const isPositiveChange = change?.direction === 'up';

  return (
    <Card
      variant="default"
      padding="md"
      className={`group ${className}`}
      {...props}
    >
      <div className="flex items-start justify-between mb-3">
        <span className="text-sm text-text-tertiary">{title}</span>
        {change && (
          <span className={`text-xs font-medium ${isPositiveChange ? 'text-success' : 'text-error'}`}>
            {isPositiveChange ? '+' : ''}{change.value}%
            {change.timeframe && (
              <span className="text-text-muted ml-1">{change.timeframe}</span>
            )}
          </span>
        )}
      </div>
      <div className="flex items-baseline gap-1">
        <span className="text-2xl font-bold text-text-primary">{value}</span>
        {unit && <span className="text-sm text-text-tertiary">{unit}</span>}
      </div>
    </Card>
  );
}

// ============================================================================
// Card Grid
// ============================================================================

export interface CardGridProps extends React.HTMLAttributes<HTMLDivElement> {
  columns?: 2 | 3 | 4;
  gap?: 'sm' | 'md' | 'lg';
}

export function CardGrid({
  columns = 3,
  gap = 'md',
  className = '',
  children,
  ...props
}: CardGridProps) {
  const gridCols = { 2: 'grid-cols-1 sm:grid-cols-2', 3: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3', 4: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4' };
  const gridGap = { sm: 'gap-3', md: 'gap-4', lg: 'gap-6' };

  return (
    <div
      className={`grid ${gridCols[columns]} ${gridGap[gap]} ${className}`}
      {...props}
    >
      {children}
    </div>
  );
}

export default Card;
