import { useState } from 'react';
import {
  Sun,
  Moon,
  Globe,
  Bell,
  Search,
  User,
  LogOut,
  Settings,
  ChevronDown,
} from 'lucide-react';

type Theme = 'dark' | 'light';
type Language = 'en' | 'vi' | 'zh';

interface HeaderProps {
  theme: Theme;
  onThemeChange: (theme: Theme) => void;
  language: Language;
  onLanguageChange: (lang: Language) => void;
}

const languages: { code: Language; label: string; flag: string }[] = [
  { code: 'en', label: 'English', flag: '🇺🇸' },
  { code: 'vi', label: 'Tiếng Việt', flag: '🇻🇳' },
  { code: 'zh', label: '中文', flag: '🇨🇳' },
];

export function Header({ theme, onThemeChange, language, onLanguageChange }: HeaderProps) {
  const [showThemeMenu, setShowThemeMenu] = useState(false);
  const [showLangMenu, setShowLangMenu] = useState(false);
  const [showUserMenu, setShowUserMenu] = useState(false);

  const currentLang = languages.find((l) => l.code === language) || languages[0];

  const toggleTheme = () => {
    onThemeChange(theme === 'dark' ? 'light' : 'dark');
  };

  return (
    <header className="h-14 bg-bg-secondary border-b border-border-subtle flex items-center justify-between px-4">
      {/* Left: Search */}
      <div className="flex items-center gap-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" />
          <input
            type="text"
            placeholder="Search..."
            className="w-64 pl-10 pr-4 py-2 bg-bg-tertiary border border-border-subtle rounded-lg text-sm text-text-primary placeholder:text-text-muted focus:border-border-focus focus:outline-none transition-colors"
          />
        </div>
      </div>

      {/* Right: Actions */}
      <div className="flex items-center gap-2">
        {/* Theme Toggle */}
        <div className="relative">
          <button
            onClick={() => {
              setShowThemeMenu(!showThemeMenu);
              setShowLangMenu(false);
              setShowUserMenu(false);
            }}
            className="p-2 rounded-lg text-text-tertiary hover:text-text-secondary hover:bg-bg-elevated transition-colors"
            title="Toggle theme"
          >
            {theme === 'dark' ? <Moon className="w-5 h-5" /> : <Sun className="w-5 h-5" />}
          </button>
          {showThemeMenu && (
            <div className="absolute right-0 mt-2 w-40 bg-bg-tertiary border border-border-subtle rounded-lg shadow-lg py-1 z-50">
              <button
                onClick={() => {
                  onThemeChange('dark');
                  setShowThemeMenu(false);
                }}
                className={`w-full flex items-center gap-2 px-4 py-2 text-sm transition-colors ${
                  theme === 'dark' ? 'text-accent-primary bg-accent-primary/10' : 'text-text-secondary hover:bg-bg-elevated'
                }`}
              >
                <Moon className="w-4 h-4" />
                Dark Mode
              </button>
              <button
                onClick={() => {
                  onThemeChange('light');
                  setShowThemeMenu(false);
                }}
                className={`w-full flex items-center gap-2 px-4 py-2 text-sm transition-colors ${
                  theme === 'light' ? 'text-accent-primary bg-accent-primary/10' : 'text-text-secondary hover:bg-bg-elevated'
                }`}
              >
                <Sun className="w-4 h-4" />
                Light Mode
              </button>
            </div>
          )}
        </div>

        {/* Language Switcher */}
        <div className="relative">
          <button
            onClick={() => {
              setShowLangMenu(!showLangMenu);
              setShowThemeMenu(false);
              setShowUserMenu(false);
            }}
            className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-text-tertiary hover:text-text-secondary hover:bg-bg-elevated transition-colors"
          >
            <Globe className="w-4 h-4" />
            <span className="text-sm">{currentLang.flag}</span>
            <ChevronDown className="w-3 h-3" />
          </button>
          {showLangMenu && (
            <div className="absolute right-0 mt-2 w-40 bg-bg-tertiary border border-border-subtle rounded-lg shadow-lg py-1 z-50">
              {languages.map((lang) => (
                <button
                  key={lang.code}
                  onClick={() => {
                    onLanguageChange(lang.code);
                    setShowLangMenu(false);
                  }}
                  className={`w-full flex items-center gap-2 px-4 py-2 text-sm transition-colors ${
                    language === lang.code ? 'text-accent-primary bg-accent-primary/10' : 'text-text-secondary hover:bg-bg-elevated'
                  }`}
                >
                  <span>{lang.flag}</span>
                  {lang.label}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Notifications */}
        <button className="relative p-2 rounded-lg text-text-tertiary hover:text-text-secondary hover:bg-bg-elevated transition-colors">
          <Bell className="w-5 h-5" />
          <span className="absolute top-1 right-1 w-2 h-2 bg-accent-primary rounded-full" />
        </button>

        {/* User Menu */}
        <div className="relative">
          <button
            onClick={() => {
              setShowUserMenu(!showUserMenu);
              setShowThemeMenu(false);
              setShowLangMenu(false);
            }}
            className="flex items-center gap-2 pl-3 pr-2 py-1.5 rounded-lg hover:bg-bg-elevated transition-colors"
          >
            <div className="w-8 h-8 rounded-full bg-accent-primary/20 flex items-center justify-center">
              <User className="w-4 h-4 text-accent-primary" />
            </div>
            <span className="text-sm font-medium text-text-primary">Admin</span>
            <ChevronDown className="w-3 h-3 text-text-muted" />
          </button>
          {showUserMenu && (
            <div className="absolute right-0 mt-2 w-48 bg-bg-tertiary border border-border-subtle rounded-lg shadow-lg py-1 z-50">
              <div className="px-4 py-2 border-b border-border-subtle">
                <p className="text-sm font-medium text-text-primary">admin@nexusai.local</p>
                <p className="text-xs text-text-muted">Administrator</p>
              </div>
              <button className="w-full flex items-center gap-2 px-4 py-2 text-sm text-text-secondary hover:bg-bg-elevated transition-colors">
                <User className="w-4 h-4" />
                Profile
              </button>
              <button className="w-full flex items-center gap-2 px-4 py-2 text-sm text-text-secondary hover:bg-bg-elevated transition-colors">
                <Settings className="w-4 h-4" />
                Settings
              </button>
              <div className="border-t border-border-subtle mt-1 pt-1">
                <button className="w-full flex items-center gap-2 px-4 py-2 text-sm text-error hover:bg-error/10 transition-colors">
                  <LogOut className="w-4 h-4" />
                  Sign Out
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
