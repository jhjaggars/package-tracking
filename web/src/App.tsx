import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Layout } from './components/layout/Layout';
import { Dashboard } from './pages/Dashboard';
import { ShipmentList } from './pages/ShipmentList';
import { ShipmentDetail } from './pages/ShipmentDetail';
import { AddShipment } from './pages/AddShipment';
import { ErrorBoundary } from './components/ErrorBoundary';

// Create a client
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      staleTime: 1 * 60 * 1000, // 1 minute
      gcTime: 5 * 60 * 1000, // 5 minutes
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Layout>
          <Routes>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={
              <ErrorBoundary>
                <Dashboard />
              </ErrorBoundary>
            } />
            <Route path="/shipments" element={
              <ErrorBoundary>
                <ShipmentList />
              </ErrorBoundary>
            } />
            <Route path="/shipments/new" element={
              <ErrorBoundary>
                <AddShipment />
              </ErrorBoundary>
            } />
            <Route path="/shipments/:id" element={
              <ErrorBoundary>
                <ShipmentDetail />
              </ErrorBoundary>
            } />
          </Routes>
        </Layout>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;