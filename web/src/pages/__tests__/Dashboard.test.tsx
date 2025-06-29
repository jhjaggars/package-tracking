import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { renderWithProviders, mockShipments } from '../../test/utils';
import { Dashboard } from '../Dashboard';
import * as api from '../../hooks/api';

// Mock the API hooks
vi.mock('../../hooks/api', () => ({
  useDashboardStats: vi.fn(),
  useShipments: vi.fn(),
}));

const mockUseDashboardStats = vi.mocked(api.useDashboardStats);
const mockUseShipments = vi.mocked(api.useShipments);

describe('Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state initially', () => {
    mockUseDashboardStats.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as any);

    mockUseShipments.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as any);

    renderWithProviders(<Dashboard />);

    // Dashboard shows greeting instead of "Package Tracking Dashboard"
    expect(screen.getByText(/Good/)).toBeInTheDocument(); // "Good morning!" or similar
    expect(screen.getAllByText('••••')).toHaveLength(4); // Loading states for stats show dots
  });

  it('renders dashboard stats correctly', async () => {
    const mockStats = {
      total_shipments: 10,
      active_shipments: 7,
      in_transit: 5,
      delivered: 3,
      requiring_attention: 2,
    };

    mockUseDashboardStats.mockReturnValue({
      data: mockStats,
      isLoading: false,
      error: null,
    } as any);

    mockUseShipments.mockReturnValue({
      data: mockShipments,
      isLoading: false,
      error: null,
    } as any);

    renderWithProviders(<Dashboard />);

    await waitFor(() => {
      expect(screen.getByText('10')).toBeInTheDocument(); // Total shipments
      expect(screen.getByText('5')).toBeInTheDocument();  // In transit
      expect(screen.getByText('3')).toBeInTheDocument();  // Delivered
      expect(screen.getByText('2')).toBeInTheDocument();  // Requiring attention
    });

    expect(screen.getByText('Total Shipments')).toBeInTheDocument();
    expect(screen.getByText('In Transit')).toBeInTheDocument();
    // Use getAllByText since 'Delivered' appears in both stats and shipment badges
    expect(screen.getAllByText('Delivered')[0]).toBeInTheDocument();
    expect(screen.getByText('Requiring Attention')).toBeInTheDocument();
  });

  it('renders recent shipments section', async () => {
    mockUseDashboardStats.mockReturnValue({
      data: { total_shipments: 3, active_shipments: 2, in_transit: 1, delivered: 1, requiring_attention: 1 },
      isLoading: false,
      error: null,
    } as any);

    mockUseShipments.mockReturnValue({
      data: mockShipments,
      isLoading: false,
      error: null,
    } as any);

    renderWithProviders(<Dashboard />);

    await waitFor(() => {
      expect(screen.getByText('Recent Activity')).toBeInTheDocument();
    });

    // Check that shipment descriptions are rendered (and sanitized)
    expect(screen.getByText('Test Package')).toBeInTheDocument();
    expect(screen.getByText('Delivered Package')).toBeInTheDocument();
    expect(screen.getByText('FedEx Package')).toBeInTheDocument();

    // Check that carriers are displayed (carrier codes are uppercase in UI)
    // Use getAllByText since there might be multiple UPS shipments
    expect(screen.getAllByText('UPS')[0]).toBeInTheDocument();
    expect(screen.getByText('FEDEX')).toBeInTheDocument();
  });

  it('renders empty state when no shipments exist', async () => {
    mockUseDashboardStats.mockReturnValue({
      data: { total_shipments: 0, active_shipments: 0, in_transit: 0, delivered: 0, requiring_attention: 0 },
      isLoading: false,
      error: null,
    } as any);

    mockUseShipments.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    } as any);

    renderWithProviders(<Dashboard />);

    await waitFor(() => {
      expect(screen.getByText('Ready to track your first package?')).toBeInTheDocument();
    });

    expect(screen.getByText('Add your tracking number and watch the magic happen as we keep you updated on every step of your delivery journey.')).toBeInTheDocument();
    expect(screen.getByText('Add Your First Shipment')).toBeInTheDocument();
  });

  it('handles error states gracefully', async () => {
    mockUseDashboardStats.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Failed to load stats'),
    } as any);

    mockUseShipments.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Failed to load shipments'),
    } as any);

    renderWithProviders(<Dashboard />);

    // When stats fail to load, the component shows 0 as default values
    await waitFor(() => {
      expect(screen.getAllByText('0')).toHaveLength(4); // All stats show 0 when data is undefined
    });
  });

  it('has working navigation links', () => {
    mockUseDashboardStats.mockReturnValue({
      data: { total_shipments: 0, active_shipments: 0, in_transit: 0, delivered: 0, requiring_attention: 0 },
      isLoading: false,
      error: null,
    } as any);

    mockUseShipments.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    } as any);

    renderWithProviders(<Dashboard />);

    const addShipmentLink = screen.getByRole('link', { name: /add shipment/i });
    expect(addShipmentLink).toHaveAttribute('href', '/shipments/new');
  });
});