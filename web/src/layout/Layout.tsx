import { useState, useEffect } from 'react';
import { Sidebar } from './Sidebar';
import { Header } from './Header';

type Theme = 'dark' | 'light';
type Language = 'en' | 'vi' | 'zh';

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [theme, setTheme] = useState<Theme>('dark');
  const [language, setLanguage] = useState<Language>('en');

  useEffect(() => {
    const savedTheme = localStorage.getItem('nexusai-theme') as Theme | null;
    if (savedTheme) {
      setTheme(savedTheme);
    }
    document.documentElement.classList.toggle('light', savedTheme === 'light');
  }, []);

  const handleThemeChange = (newTheme: Theme) => {
    setTheme(newTheme);
    localStorage.setItem('nexusai-theme', newTheme);
    document.documentElement.classList.toggle('light', newTheme === 'light');
  };

  return (
    <div className={`flex h-screen bg-bg-primary ${theme === 'light' ? 'light' : 'dark'}`}>
      <Sidebar collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)} />
      <div className="flex-1 flex flex-col overflow-hidden">
        <Header
          theme={theme}
          onThemeChange={handleThemeChange}
          language={language}
          onLanguageChange={setLanguage}
        />
        <main className="flex-1 overflow-auto p-6 bg-bg-primary">
          {children}
        </main>
      </div>
    </div>
  );
}
