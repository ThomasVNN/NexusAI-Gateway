import React from 'react';

export type ActivityType = 
  | 'api_call'
  | 'key_created'
  | 'key_revoked'
  | 'provider_added'
  | 'provider_updated'
  | 'error'
  | 'warning'
  | 'info'
  | 'user_action';

export interface ActivityItem {
  id: string;
  type: ActivityType;
  title: string;
  description?: string;
  timestamp: string | Date;
  metadata?: Record<string, string | number | boolean>;
  actor?: {
    name: string;
    avatar?: string;
  };
}

interface ActivityTimelineProps {
  activities: ActivityItem[];
  title?: string;
  maxItems?: number;
  onItemClick?: (activity: ActivityItem) => void;
  className?: string;
}

const typeConfig: Record<ActivityType, {
  icon: React.ReactNode;
  color: string;
  bgColor: string;
  borderColor: string;
}> = {
  api_call: {
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
    ),
    color: 'text-accent-primary',
    bgColor: 'bg-accent-primary/10',
    borderColor: 'border-l-accent-primary',
  },
  key_created: {
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
      </svg>
    ),
    color: 'text-success',
    bgColor: 'bg-success/10',
    borderColor: 'border-l-success',
  },
  key_revoked: {
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M18.364 18.364A9 9 0 005.636 5.636m12.728 12.728A9 9 0 015.636 5.636m12.728 12.728L5.636 5.636" />
      </svg>
    ),
    color: 'text-error',
    bgColor: 'bg-error/10',
    borderColor: 'border-l-error',
  },
  provider_added: {
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
      </svg>
    ),
    color: 'text-info',
    bgColor: 'bg-info/10',
    borderColor: 'border-l-info',
  },
  provider_updated: {
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
      </svg>
    ),
    color: 'text-warning',
    bgColor: 'bg-warning/10',
    borderColor: 'border-l-warning',
  },
  error: {
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    color: 'text-error',
    bgColor: 'bg-error/10',
    borderColor: 'border-l-error',
  },
  warning: {
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
    ),
    color: 'text-warning',
    bgColor: 'bg-warning/10',
    borderColor: 'border-l-warning',
  },
  info: {
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    color: 'text-info',
    bgColor: 'bg-info/10',
    borderColor: 'border-l-info',
  },
  user_action: {
    icon: (
      <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
    ),
    color: 'text-text-secondary',
    bgColor: 'bg-bg-elevated',
    borderColor: 'border-l-text-secondary',
  },
};

function formatTimestamp(timestamp: string | Date): { relative: string; absolute: string } {
  const date = typeof timestamp === 'string' ? new Date(timestamp) : timestamp;
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  let relative: string;
  if (diffMins < 1) {
    relative = 'Just now';
  } else if (diffMins < 60) {
    relative = `${diffMins}m ago`;
  } else if (diffHours < 24) {
    relative = `${diffHours}h ago`;
  } else if (diffDays < 7) {
    relative = `${diffDays}d ago`;
  } else {
    relative = date.toLocaleDateString();
  }

  const absolute = date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });

  return { relative, absolute };
}

function ActivityItemComponent({ 
  activity, 
  isLast,
  onClick 
}: { 
  activity: ActivityItem; 
  isLast: boolean;
  onClick?: (activity: ActivityItem) => void;
}) {
  const config = typeConfig[activity.type];
  const { relative, absolute } = formatTimestamp(activity.timestamp);

  return (
    <div 
      className={`
        relative pl-8 pb-6
        ${!isLast ? 'border-l-2 border-border-subtle' : ''}
        last:pb-0 last:border-l-transparent
      `}
    >
      {/* Timeline Dot */}
      <div 
        className={`
          absolute left-0 top-0 
          -translate-x-1/2 -translate-y-1
          w-8 h-8 rounded-full
          flex items-center justify-center
          ${config.bgColor} ${config.color}
          border-2 border-bg-primary
          shadow-sm
        `}
      >
        {config.icon}
      </div>

      {/* Content */}
      <div 
        className={`
          group
          ${onClick ? 'cursor-pointer' : ''}
          hover:bg-bg-elevated/50
          -ml-4 pl-4 pr-4 py-3 rounded-lg
          transition-all duration-150
        `}
        onClick={() => onClick?.(activity)}
        role={onClick ? 'button' : undefined}
        tabIndex={onClick ? 0 : undefined}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="flex-1 min-w-0">
            {/* Actor & Title */}
            <div className="flex items-center gap-2 flex-wrap">
              {activity.actor && (
                <span className="text-sm font-medium text-text-secondary">
                  {activity.actor.name}
                </span>
              )}
              <h4 className="text-sm font-medium text-text-primary">
                {activity.title}
              </h4>
            </div>

            {/* Description */}
            {activity.description && (
              <p className="mt-1 text-sm text-text-tertiary line-clamp-2">
                {activity.description}
              </p>
            )}

            {/* Metadata */}
            {activity.metadata && Object.keys(activity.metadata).length > 0 && (
              <div className="mt-2 flex flex-wrap gap-2">
                {Object.entries(activity.metadata).map(([key, value]) => (
                  <span 
                    key={key}
                    className="inline-flex items-center px-2 py-0.5 rounded text-xs font-mono bg-bg-muted text-text-tertiary"
                  >
                    {key}: {String(value)}
                  </span>
                ))}
              </div>
            )}
          </div>

          {/* Timestamp */}
          <div className="flex-shrink-0 text-right">
            <span className="text-xs text-text-tertiary" title={absolute}>
              {relative}
            </span>
          </div>
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
          <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      </div>
      <p className="text-sm text-text-tertiary">No activity yet</p>
      <p className="text-xs text-text-muted mt-1">Recent events will appear here</p>
    </div>
  );
}

export function ActivityTimeline({
  activities,
  title = 'Recent Activity',
  maxItems,
  onItemClick,
  className = '',
}: ActivityTimelineProps) {
  const displayedActivities = maxItems ? activities.slice(0, maxItems) : activities;

  return (
    <div className={`panel ${className}`}>
      {/* Header */}
      <div className="panel-header">
        <div className="flex items-center gap-2">
          <svg className="w-5 h-5 text-text-tertiary" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <h3 className="text-lg font-semibold text-text-primary">{title}</h3>
        </div>
        {activities.length > 0 && (
          <span className="text-xs text-text-tertiary">
            {activities.length} total
          </span>
        )}
      </div>

      {/* Timeline */}
      <div className="p-5">
        {displayedActivities.length === 0 ? (
          <EmptyState />
        ) : (
          <div className="space-y-0">
            {displayedActivities.map((activity, index) => (
              <ActivityItemComponent
                key={activity.id}
                activity={activity}
                isLast={index === displayedActivities.length - 1}
                onClick={onItemClick}
              />
            ))}
          </div>
        )}
      </div>

      {/* Footer */}
      {activities.length > (maxItems || 0) && (
        <div className="px-5 py-3 border-t border-border-subtle">
          <button className="text-sm text-text-tertiary hover:text-accent-primary transition-colors flex items-center gap-1">
            <span>View all activity</span>
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      )}
    </div>
  );
}

export default ActivityTimeline;
