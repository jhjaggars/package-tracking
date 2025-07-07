import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { Suspense, lazy } from 'react';
import { Layout } from './components/layout/Layout';
import { ErrorBoundary } from './components/ErrorBoundary';
// Simple loading component for Suspense fallback
const LoadingFallback = () => (
  <div className="flex items-center justify-center min-h-[200px]">
    <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-r-primary" />
  </div>
);
import { ThemeProvider } from './contexts/ThemeContext';

// Lazy load page components for code splitting
const Dashboard = lazy(() => import('./pages/Dashboard').then(module => ({ default: module.Dashboard })));
const ShipmentList = lazy(() => import('./pages/ShipmentList').then(module => ({ default: module.ShipmentList })));
const ShipmentDetail = lazy(() => import('./pages/ShipmentDetail').then(module => ({ default: module.ShipmentDetail })));
const AddShipment = lazy(() => import('./pages/AddShipment').then(module => ({ default: module.AddShipment })));

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
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          <Layout>
            <Suspense fallback={<LoadingFallback />}>
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
            </Suspense>
          </Layout>
        </BrowserRouter>
      </QueryClientProvider>
    </ThemeProvider>
  );
}

export default App;