import type { ReactElement } from 'react';
import { render } from '@testing-library/react';
import type { RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import type { Shipment, TrackingEvent, Carrier } from '../types/api';

// Create a custom render function that includes providers
function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

interface CustomRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  queryClient?: QueryClient;
  initialRoute?: string;
}

export function renderWithProviders(
  ui: ReactElement,
  options: CustomRenderOptions = {}
) {
  const { queryClient = createTestQueryClient(), initialRoute = '/', ...renderOptions } = options;

  function Wrapper({ children }: { children: React.ReactNode }) {
    // Set initial route if provided
    if (initialRoute !== '/') {
      window.history.pushState({}, 'Test page', initialRoute);
    }

    return (
      <QueryClientProvider client={queryClient}>
        <BrowserRouter>
          {children}
        </BrowserRouter>
      </QueryClientProvider>
    );
  }

  return {
    ...render(ui, { wrapper: Wrapper, ...renderOptions }),
    queryClient,
  };
}

// Mock data factories for testing
export const mockShipment: Shipment = {
  id: 1,
  tracking_number: '1Z999AA1234567890',
  carrier: 'ups',
  description: 'Test Package',
  status: 'in_transit',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T12:00:00Z',
  is_delivered: false,
  manual_refresh_count: 0,
};

export const mockDeliveredShipment: Shipment = {
  ...mockShipment,
  id: 2,
  status: 'delivered',
  is_delivered: true,
  description: 'Delivered Package',
};

export const mockTrackingEvent: TrackingEvent = {
  id: 1,
  shipment_id: 1,
  status: 'in_transit',
  description: 'Package is in transit',
  location: 'Chicago, IL',
  timestamp: '2024-01-01T12:00:00Z',
  created_at: '2024-01-01T12:00:00Z',
};

export const mockCarrier: Carrier = {
  id: 1,
  name: 'UPS',
  code: 'ups',
  api_endpoint: 'https://api.ups.com',
  active: true,
};

export const mockShipments: Shipment[] = [
  mockShipment,
  mockDeliveredShipment,
  {
    ...mockShipment,
    id: 3,
    carrier: 'fedex',
    status: 'pending',
    description: 'FedEx Package',
  },
];

// Helper to create mock API responses
export function createMockApiResponse<T>(data: T) {
  return Promise.resolve({ data });
}

// Helper to create mock API error
export function createMockApiError(message = 'Test error', status = 500) {
  const error = new Error(message) as any;
  error.response = {
    status,
    data: { message },
  };
  return Promise.reject(error);
}

// Re-export everything from testing library
export * from '@testing-library/react';
export { userEvent } from '@testing-library/user-event';