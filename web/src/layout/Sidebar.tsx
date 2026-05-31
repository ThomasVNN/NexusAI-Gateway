import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  LayoutDashboard,
  Key,
  Network,
  Users,
  CreditCard,
  BarChart3,
  Settings,
  ChevronDown,
  ChevronRight,
  Sparkles,
} from 'lucide-react';

interface NavItem {
  id: string;
  label: string;
  icon: React.ReactNode;
  path?: string;
  children?: NavItem[];
}

const navSections: NavItem[] = [
  {
    id: 'overview',
    label: 'Overview',
    icon: <LayoutDashboard className="w-5 h-5" />,
    children: [
      { id: 'dashboard', label: 'Dashboard', icon: <LayoutDashboard className="w-4 h-4" />, path: '/' },
      { id: 'analytics', label: 'Analytics', icon: <BarChart3 className="w-4 h-4" />, path: '/analytics' },
    ],
  },
  {
    id: 'management',
    label: 'Management',
    icon: <Key className="w-5 h-5" />,
    children: [
      { id: 'channels', label: 'Channels', icon: <Network className="w-4 h-4" />, path: '/channels' },
      { id: 'tokens', label: 'API Keys', icon: <Key className="w-4 h-4" />, path: '/tokens' },
      { id: 'users', label: 'Users', icon: <Users className="w-4 h-4" />, path: '/users' },
    ],
  },
  {
    id: 'billing',
    label: 'Billing',
    icon: <CreditCard className="w-5 h-5" />,
    children: [
      { id: 'billing', label: 'Billing & Usage', icon: <CreditCard className="w-4 h-4" />, path: '/billing' },
      { id: 'topup', label: 'Top-Up', icon: <CreditCard className="w-4 h-4" />, path: '/topup' },
    ],
  },
  {
    id: 'system',
    label: 'System',
    icon: <Settings className="w-5 h-5" />,
    children: [
      { id: 'settings', label: 'Settings', icon: <Settings className="w-4 h-4" />, path: '/settings' },
    ],
  },
];

interface SidebarProps {
  collapsed?: boolean;
  onToggle?: () => void;
}

export function Sidebar({ collapsed = false, onToggle }: SidebarProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const [expandedSections, setExpandedSections] = useState<string[]>(['overview', 'management']);

  const toggleSection = (sectionId: string) => {
    setExpandedSections((prev) =>
      prev.includes(sectionId)
        ? prev.filter((id) => id !== sectionId)
        : [...prev, sectionId]
    );
  };

  const isActive = (path: string) => location.pathname === path;

  const renderNavItem = (item: NavItem, depth = 0) => {
    if (item.children) {
      const isExpanded = expandedSections.includes(item.id);
      return (
        <div key={item.id}>
          <button
            onClick={() => toggleSection(item.id)}
            className={`w-full flex items-center justify-between px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
              depth === 0
                ? 'text-text-secondary hover:text-text-primary hover:bg-bg-elevated'
                : 'text-text-tertiary hover:text-text-secondary hover:bg-bg-elevated/50 ml-2'
            }`}
            style={{ paddingLeft: depth === 0 ? '0.75rem' : `${1 + depth * 0.5}rem` }}
          >
            <div className="flex items-center gap-2">
              {item.icon}
              {!collapsed && <span>{item.label}</span>}
            </div>
            {!collapsed && (
              isExpanded ? (
                <ChevronDown className="w-4 h-4" />
              ) : (
                <ChevronRight className="w-4 h-4" />
              )
            )}
          </button>
          {!collapsed && isExpanded && (
            <div className="mt-1 space-y-0.5">
              {item.children.map((child) => renderNavItem(child, depth + 1))}
            </div>
          )}
        </div>
      );
    }

    return (
      <button
        key={item.id}
        onClick={() => item.path && navigate(item.path)}
        className={`w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
          isActive(item.path || '')
            ? 'bg-accent-primary/10 text-accent-primary border-l-2 border-accent-primary'
            : 'text-text-tertiary hover:text-text-secondary hover:bg-bg-elevated/50'
        }`}
        style={{ paddingLeft: depth === 0 ? '0.75rem' : `${1 + depth * 0.5 + 0.5}rem` }}
      >
        {item.icon}
        {!collapsed && <span>{item.label}</span>}
      </button>
    );
  };

  return (
    <aside
      className={`h-full bg-bg-secondary border-r border-border-subtle flex flex-col transition-all duration-300 ${
        collapsed ? 'w-16' : 'w-64'
      }`}
    >
      {/* Logo */}
      <div className="p-4 border-b border-border-subtle">
        <div className="flex items-center gap-3">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-accent-primary to-accent-secondary flex items-center justify-center shrink-0">
            <Sparkles className="w-4 h-4 text-white" />
          </div>
          {!collapsed && (
            <div className="overflow-hidden">
              <div className="flex items-center gap-2">
                <h1 className="font-bold text-text-primary whitespace-nowrap">NexusAI</h1>
                <span className="px-1.5 py-0.5 text-2xs font-medium bg-accent-primary/20 text-accent-primary rounded">
                  v2
                </span>
              </div>
              <p className="text-2xs text-text-tertiary">Gateway Control</p>
            </div>
          )}
        </div>
      </div>

      {/* Navigation */}
      <nav className="flex-1 p-3 space-y-4 overflow-y-auto">
        {navSections.map((section) => (
          <div key={section.id}>
            {!collapsed && (
              <h3 className="px-3 mb-2 text-2xs font-semibold text-text-muted uppercase tracking-wider">
                {section.label}
              </h3>
            )}
            {renderNavItem(section)}
          </div>
        ))}
      </nav>

      {/* Collapse Toggle */}
      <div className="p-3 border-t border-border-subtle">
        <button
          onClick={onToggle}
          className="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg text-sm text-text-tertiary hover:text-text-secondary hover:bg-bg-elevated transition-colors"
        >
          {collapsed ? (
            <ChevronRight className="w-4 h-4" />
          ) : (
            <>
              <ChevronDown className="w-4 h-4 rotate-90" />
              <span>Collapse</span>
            </>
          )}
        </button>
      </div>
    </aside>
  );
}
