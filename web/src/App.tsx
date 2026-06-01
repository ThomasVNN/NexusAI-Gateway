import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { Layout } from './layout/Layout';
import { DashboardPage } from './pages/DashboardPage';
import { AnalyticsPage } from './pages/AnalyticsPage';
import { ChannelsPage } from './pages/ChannelsPage';
import { CombosPage } from './pages/CombosPage';
import { TokensPage } from './pages/TokensPage';
import { UsersPage } from './pages/UsersPage';
import { BillingPage } from './pages/BillingPage';
import { TopUpPage } from './pages/TopUpPage';
import { SettingsPage } from './pages/SettingsPage';
import { ProvidersPage } from './pages/ProvidersPage';
import HealthPage from './pages/HealthPage';

function App() {
  return (
    <BrowserRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<DashboardPage />} />
          <Route path="/analytics" element={<AnalyticsPage />} />
          <Route path="/channels" element={<ChannelsPage />} />
          <Route path="/combos" element={<CombosPage />} />
          <Route path="/tokens" element={<TokensPage />} />
          <Route path="/users" element={<UsersPage />} />
          <Route path="/billing" element={<BillingPage />} />
          <Route path="/topup" element={<TopUpPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/health" element={<HealthPage />} />
          <Route path="/providers" element={<ProvidersPage />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  );
}

export default App;
