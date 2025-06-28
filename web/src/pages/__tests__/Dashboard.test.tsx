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

    expect(screen.getByText('Package Tracking Dashboard')).toBeInTheDocument();
    expect(screen.getAllByText('...')).toHaveLength(4); // Loading states for stats
  });

  it('renders dashboard stats correctly', async () => {
    const mockStats = {
      total: 10,
      in_transit: 5,
      delivered: 3,
      pending: 2,
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
      expect(screen.getByText('10')).toBeInTheDocument(); // Total
      expect(screen.getByText('5')).toBeInTheDocument();  // In transit
      expect(screen.getByText('3')).toBeInTheDocument();  // Delivered
      expect(screen.getByText('2')).toBeInTheDocument();  // Pending
    });

    expect(screen.getByText('Total Shipments')).toBeInTheDocument();
    expect(screen.getByText('In Transit')).toBeInTheDocument();
    expect(screen.getByText('Delivered')).toBeInTheDocument();
    expect(screen.getByText('Pending')).toBeInTheDocument();
  });

  it('renders recent shipments section', async () => {
    mockUseDashboardStats.mockReturnValue({
      data: { total: 3, in_transit: 1, delivered: 1, pending: 1 },
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
      expect(screen.getByText('Recent Shipments')).toBeInTheDocument();
    });

    // Check that shipment descriptions are rendered (and sanitized)
    expect(screen.getByText('Test Package')).toBeInTheDocument();
    expect(screen.getByText('Delivered Package')).toBeInTheDocument();
    expect(screen.getByText('FedEx Package')).toBeInTheDocument();

    // Check that carriers are displayed
    expect(screen.getByText('UPS')).toBeInTheDocument();
    expect(screen.getByText('FEDEX')).toBeInTheDocument();
  });

  it('renders empty state when no shipments exist', async () => {
    mockUseDashboardStats.mockReturnValue({
      data: { total: 0, in_transit: 0, delivered: 0, pending: 0 },
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
      expect(screen.getByText('No shipments yet')).toBeInTheDocument();
    });

    expect(screen.getByText('Get started by adding your first shipment to track.')).toBeInTheDocument();
    expect(screen.getByText('Add First Shipment')).toBeInTheDocument();
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

    await waitFor(() => {
      expect(screen.getByText('--')).toBeInTheDocument(); // Error state for stats
    });
  });

  it('has working navigation links', () => {
    mockUseDashboardStats.mockReturnValue({
      data: { total: 0, in_transit: 0, delivered: 0, pending: 0 },
      isLoading: false,
      error: null,
    } as any);

    mockUseShipments.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
    } as any);

    renderWithProviders(<Dashboard />);

    const addShipmentLink = screen.getByRole('link', { name: /add first shipment/i });
    expect(addShipmentLink).toHaveAttribute('href', '/shipments/new');
  });
});