import React from 'react';

export interface Insight {
  id: string;
  type: 'recommendation' | 'warning' | 'success' | 'info';
  title: string;
  description: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  timestamp?: string;
}

interface AIInsightsPanelProps {
  insights: Insight[];
  title?: string;
  isLoading?: boolean;
  onDismiss?: (id: string) => void;
  className?: string;
}

const typeConfig = {
  recommendation: {
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
      </svg>
    ),
    color: 'accent',
    bgColor: 'bg-accent-primary/10',
    borderColor: 'border-accent-primary/30',
    iconColor: 'text-accent-primary',
  },
  warning: {
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
    ),
    color: 'warning',
    bgColor: 'bg-warning/10',
    borderColor: 'border-warning/30',
    iconColor: 'text-warning',
  },
  success: {
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    color: 'success',
    bgColor: 'bg-success/10',
    borderColor: 'border-success/30',
    iconColor: 'text-success',
  },
  info: {
    icon: (
      <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
    color: 'info',
    bgColor: 'bg-info/10',
    borderColor: 'border-info/30',
    iconColor: 'text-info',
  },
};

function InsightCard({ insight, onDismiss }: { insight: Insight; onDismiss?: (id: string) => void }) {
  const config = typeConfig[insight.type];

  return (
    <div
      className={`
        relative overflow-hidden
        ${config.bgColor} ${config.borderColor}
        border rounded-lg p-4
        transition-all duration-200
        hover:shadow-md
        animate-slide-up
      `}
    >
      <div className="flex gap-3">
        {/* Icon */}
        <div className={`flex-shrink-0 ${config.iconColor}`}>
          {config.icon}
        </div>

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="flex items-start justify-between gap-2">
            <h4 className="text-sm font-semibold text-text-primary">
              {insight.title}
            </h4>
            {onDismiss && (
              <button
                onClick={() => onDismiss(insight.id)}
                className="flex-shrink-0 p-1 rounded hover:bg-bg-elevated text-text-tertiary hover:text-text-secondary transition-colors"
                aria-label="Dismiss insight"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            )}
          </div>

          <p className="mt-1 text-sm text-text-secondary leading-relaxed">
            {insight.description}
          </p>

          <div className="mt-3 flex items-center justify-between gap-3">
            {insight.timestamp && (
              <span className="text-xs text-text-tertiary">
                {insight.timestamp}
              </span>
            )}
            {insight.action && (
              <button
                onClick={insight.action.onClick}
                className={`
                  text-sm font-medium
                  ${config.iconColor}
                  hover:underline
                  transition-all
                `}
              >
                {insight.action.label}
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function SkeletonInsight() {
  return (
    <div className="bg-bg-elevated/50 rounded-lg p-4 animate-pulse">
      <div className="flex gap-3">
        <div className="w-5 h-5 rounded bg-bg-muted" />
        <div className="flex-1 space-y-2">
          <div className="h-4 w-2/3 bg-bg-muted rounded" />
          <div className="h-3 w-full bg-bg-muted rounded" />
          <div className="h-3 w-4/5 bg-bg-muted rounded" />
        </div>
      </div>
    </div>
  );
}

export function AIInsightsPanel({
  insights,
  title = 'AI Insights',
  isLoading = false,
  onDismiss,
  className = '',
}: AIInsightsPanelProps) {
  return (
    <div className={`panel ${className}`}>
      {/* Header */}
      <div className="panel-header">
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-accent-primary to-accent-secondary flex items-center justify-center">
            <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
            </svg>
          </div>
          <h3 className="text-lg font-semibold text-text-primary">{title}</h3>
        </div>
        <span className="badge badge-info">
          {insights.length} {insights.length === 1 ? 'insight' : 'insights'}
        </span>
      </div>

      {/* Content */}
      <div className="p-5 space-y-3">
        {isLoading ? (
          <>
            <SkeletonInsight />
            <SkeletonInsight />
            <SkeletonInsight />
          </>
        ) : insights.length === 0 ? (
          <div className="text-center py-8">
            <div className="w-12 h-12 mx-auto mb-3 rounded-full bg-bg-elevated flex items-center justify-center">
              <svg className="w-6 h-6 text-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
              </svg>
            </div>
            <p className="text-sm text-text-tertiary">No insights available</p>
            <p className="text-xs text-text-muted mt-1">AI analysis will appear here</p>
          </div>
        ) : (
          insights.map((insight, index) => (
            <div
              key={insight.id}
              style={{ animationDelay: `${index * 50}ms`, animationFillMode: 'both' }}
            >
              <InsightCard insight={insight} onDismiss={onDismiss} />
            </div>
          ))
        )}
      </div>

      {/* Footer */}
      {insights.length > 0 && (
        <div className="px-5 py-3 border-t border-border-subtle">
          <button className="text-sm text-text-tertiary hover:text-accent-primary transition-colors flex items-center gap-1">
            <span>View all insights</span>
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
            </svg>
          </button>
        </div>
      )}
    </div>
  );
}

export default AIInsightsPanel;
