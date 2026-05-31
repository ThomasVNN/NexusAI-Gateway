import React from 'react';

export interface WorkspaceOverviewItem {
  id: string;
  name: string;
  status: 'healthy' | 'degraded' | 'down' | 'maintenance';
  type: 'gateway' | 'model' | 'provider' | 'service';
  metrics?: {
    label: string;
    value: string | number;
    status?: 'good' | 'warning' | 'critical';
  }[];
  lastActivity?: string;
  href?: string;
}

interface WorkspaceOverviewProps {
  items: WorkspaceOverviewItem[];
  title?: string;
  isLoading?: boolean;
  onItemClick?: (item: WorkspaceOverviewItem) => void;
  className?: string;
}

const statusConfig = {
  healthy: {
    label: 'Healthy',
    color: 'text-success',
    bgColor: 'bg-success/10',
    dotColor: 'bg-success',
  },
  degraded: {
    label: 'Degraded',
    color: 'text-warning',
    bgColor: 'bg-warning/10',
    dotColor: 'bg-warning',
  },
  down: {
    label: 'Down',
    color: 'text-error',
    bgColor: 'bg-error/10',
    dotColor: 'bg-error',
  },
  maintenance: {
    label: 'Maintenance',
    color: 'text-info',
    bgColor: 'bg-info/10',
    dotColor: 'bg-info',
  },
};

const typeIcons = {
  gateway: (
    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
    </svg>
  ),
  model: (
    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
    </svg>
  ),
  provider: (
    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M21 12a9 9 0 01-9 9m9-9a9 9 0 00-9-9m9 9H3m9 9a9 9 0 01-9-9m9 9c1.657 0 3-4.03 3-9s-1.343-9-3-9m0 18c-1.657 0-3-4.03-3-9s1.343-9 3-9m-9 9a9 9 0 019-9" />
    </svg>
  ),
  service: (
    <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z" />
    </svg>
  ),
};

function WorkspaceItemCard({ 
  item, 
  onClick 
}: { 
  item: WorkspaceOverviewItem;
  onClick?: (item: WorkspaceOverviewItem) => void;
}) {
  const status = statusConfig[item.status];
  const icon = typeIcons[item.type];

  return (
    <div
      className={`
        group relative
        bg-bg-tertiary border border-border-subtle rounded-lg
        p-4
        transition-all duration-200
        hover:border-border-default hover:shadow-md
        ${onClick ? 'cursor-pointer' : ''}
        animate-fade-in
      `}
      onClick={() => onClick?.(item)}
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
    >
      {/* Status Indicator */}
      <div className="absolute top-3 right-3">
        <div className="flex items-center gap-1.5">
          <div className={`w-2 h-2 rounded-full ${status.dotColor} ${item.status === 'healthy' ? 'animate-pulse' : ''}`} />
          <span className={`text-xs font-medium ${status.color}`}>
            {status.label}
          </span>
        </div>
      </div>

      {/* Header */}
      <div className="flex items-start gap-3 mb-3">
        <div className={`flex-shrink-0 w-10 h-10 rounded-lg ${status.bgColor} flex items-center justify-center ${status.color}`}>
          {icon}
        </div>
        <div className="flex-1 min-w-0">
          <h4 className="text-sm font-semibold text-text-primary truncate">
            {item.name}
          </h4>
          <span className="text-xs text-text-tertiary capitalize">
            {item.type}
          </span>
        </div>
      </div>

      {/* Metrics */}
      {item.metrics && item.metrics.length > 0 && (
        <div className="space-y-2">
          {item.metrics.map((metric, index) => {
            const metricColor = 
              metric.status === 'critical' ? 'text-error' :
              metric.status === 'warning' ? 'text-warning' :
              'text-text-secondary';
            
            return (
              <div key={index} className="flex items-center justify-between">
                <span className="text-xs text-text-tertiary">{metric.label}</span>
                <span className={`text-xs font-mono font-medium ${metricColor}`}>
                  {typeof metric.value === 'number' ? metric.value.toLocaleString() : metric.value}
                </span>
              </div>
            );
          })}
        </div>
      )}

      {/* Last Activity */}
      {item.lastActivity && (
        <div className="mt-3 pt-3 border-t border-border-subtle">
          <span className="text-xs text-text-muted">
            Last active: {item.lastActivity}
          </span>
        </div>
      )}

      {/* Hover Arrow */}
      {onClick && (
        <div className="absolute bottom-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity">
          <svg className="w-4 h-4 text-text-tertiary" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
          </svg>
        </div>
      )}
    </div>
  );
}

function SkeletonItem() {
  return (
    <div className="bg-bg-tertiary border border-border-subtle rounded-lg p-4 animate-pulse">
      <div className="flex items-start gap-3 mb-3">
        <div className="w-10 h-10 rounded-lg bg-bg-elevated" />
        <div className="flex-1">
          <div className="h-4 w-2/3 bg-bg-elevated rounded mb-2" />
          <div className="h-3 w-1/3 bg-bg-elevated rounded" />
        </div>
      </div>
      <div className="space-y-2">
        <div className="flex justify-between">
          <div className="h-3 w-1/3 bg-bg-elevated rounded" />
          <div className="h-3 w-1/4 bg-bg-elevated rounded" />
        </div>
        <div className="flex justify-between">
          <div className="h-3 w-1/4 bg-bg-elevated rounded" />
          <div className="h-3 w-1/3 bg-bg-elevated rounded" />
        </div>
      </div>
    </div>
  );
}

function EmptyState() {
  return (
    <div className="text-center py-8">
      <div className="w-12 h-12 mx-auto mb-3 rounded-full bg-bg-elevated flex items-center justify-center">
        <svg className="w-6 h-6 text-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
        </svg>
      </div>
      <p className="text-sm text-text-tertiary">No workspaces configured</p>
      <p className="text-xs text-text-muted mt-1">Add providers to get started</p>
    </div>
  );
}

export function WorkspaceOverview({
  items,
  title = 'Workspace Overview',
  isLoading = false,
  onItemClick,
  className = '',
}: WorkspaceOverviewProps) {
  return (
    <div className={`panel ${className}`}>
      {/* Header */}
      <div className="panel-header">
        <div className="flex items-center gap-2">
          <svg className="w-5 h-5 text-text-tertiary" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <h3 className="text-lg font-semibold text-text-primary">{title}</h3>
        </div>
        <div className="flex items-center gap-2 text-xs text-text-tertiary">
          <span className="flex items-center gap-1">
            <div className="w-2 h-2 rounded-full bg-success" />
            {items.filter(i => i.status === 'healthy').length} healthy
          </span>
          {items.filter(i => i.status !== 'healthy').length > 0 && (
            <span className="flex items-center gap-1">
              <div className="w-2 h-2 rounded-full bg-warning" />
              {items.filter(i => i.status !== 'healthy').length} issues
            </span>
          )}
        </div>
      </div>

      {/* Content */}
      <div className="p-5">
        {isLoading ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <SkeletonItem />
            <SkeletonItem />
            <SkeletonItem />
            <SkeletonItem />
          </div>
        ) : items.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            {items.map((item, index) => (
              <div
                key={item.id}
                style={{ animationDelay: `${index * 50}ms`, animationFillMode: 'both' }}
              >
                <WorkspaceItemCard item={item} onClick={onItemClick} />
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      {items.length > 0 && (
        <div className="px-5 py-3 border-t border-border-subtle">
          <button className="text-sm text-text-tertiary hover:text-accent-primary transition-colors flex items-center gap-1">
            <span>Manage workspaces</span>
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      )}
    </div>
  );
}

export default WorkspaceOverview;
