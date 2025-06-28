import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { renderWithProviders, mockShipments } from '../../test/utils';
import { ShipmentList } from '../ShipmentList';
import * as api from '../../hooks/api';

// Mock the API hooks
vi.mock('../../hooks/api', () => ({
  useShipments: vi.fn(),
}));

const mockUseShipments = vi.mocked(api.useShipments);

describe('ShipmentList', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state initially', () => {
    mockUseShipments.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
      refetch: vi.fn(),
    } as any);

    renderWithProviders(<ShipmentList />);

    expect(screen.getByText('Loading shipments...')).toBeInTheDocument();
  });

  it('renders shipments table correctly', async () => {
    mockUseShipments.mockReturnValue({
      data: mockShipments,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    } as any);

    renderWithProviders(<ShipmentList />);

    await waitFor(() => {
      expect(screen.getByText('Shipments')).toBeInTheDocument();
    });

    // Check table headers
    expect(screen.getByText('Description')).toBeInTheDocument();
    expect(screen.getByText('Tracking Number')).toBeInTheDocument();
    expect(screen.getByText('Carrier')).toBeInTheDocument();
    expect(screen.getByText('Status')).toBeInTheDocument();
    expect(screen.getByText('Actions')).toBeInTheDocument();

    // Check shipment data (sanitized descriptions)
    expect(screen.getByText('Test Package')).toBeInTheDocument();
    expect(screen.getByText('Delivered Package')).toBeInTheDocument();
    expect(screen.getByText('FedEx Package')).toBeInTheDocument();

    // Check tracking numbers
    expect(screen.getByText('1Z999AA1234567890')).toBeInTheDocument();

    // Check carriers
    expect(screen.getByText('ups')).toBeInTheDocument();
    expect(screen.getByText('fedex')).toBeInTheDocument();
  });

  it('handles search functionality', async () => {
    const user = userEvent.setup();
    
    mockUseShipments.mockReturnValue({
      data: mockShipments,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    } as any);

    renderWithProviders(<ShipmentList />);

    await waitFor(() => {
      expect(screen.getByText('Test Package')).toBeInTheDocument();
    });

    // Search for specific shipment
    const searchInput = screen.getByPlaceholderText('Search shipments...');
    await user.type(searchInput, 'FedEx');

    // Should only show FedEx package
    expect(screen.getByText('FedEx Package')).toBeInTheDocument();
    expect(screen.queryByText('Test Package')).not.toBeInTheDocument();
    expect(screen.queryByText('Delivered Package')).not.toBeInTheDocument();
  });

  it('handles carrier filtering', async () => {
    const user = userEvent.setup();
    
    mockUseShipments.mockReturnValue({
      data: mockShipments,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    } as any);

    renderWithProviders(<ShipmentList />);

    await waitFor(() => {
      expect(screen.getByText('Test Package')).toBeInTheDocument();
    });

    // Filter by UPS
    const carrierFilter = screen.getByDisplayValue('all');
    await user.selectOptions(carrierFilter, 'ups');

    // Should only show UPS packages
    expect(screen.getByText('Test Package')).toBeInTheDocument();
    expect(screen.getByText('Delivered Package')).toBeInTheDocument();
    expect(screen.queryByText('FedEx Package')).not.toBeInTheDocument();
  });

  it('handles status filtering', async () => {
    const user = userEvent.setup();
    
    mockUseShipments.mockReturnValue({
      data: mockShipments,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    } as any);

    renderWithProviders(<ShipmentList />);

    await waitFor(() => {
      expect(screen.getByText('Test Package')).toBeInTheDocument();
    });

    // Filter by delivered status
    const statusFilter = screen.getByDisplayValue('all');
    await user.selectOptions(statusFilter, 'delivered');

    // Should only show delivered packages
    expect(screen.getByText('Delivered Package')).toBeInTheDocument();
    expect(screen.queryByText('Test Package')).not.toBeInTheDocument();
    expect(screen.queryByText('FedEx Package')).not.toBeInTheDocument();
  });

  it('renders empty state when no shipments', async () => {
    mockUseShipments.mockReturnValue({
      data: [],
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    } as any);

    renderWithProviders(<ShipmentList />);

    await waitFor(() => {
      expect(screen.getByText('No shipments found')).toBeInTheDocument();
    });

    expect(screen.getByText('Add your first shipment to get started.')).toBeInTheDocument();
    expect(screen.getByText('Add Shipment')).toBeInTheDocument();
  });

  it('handles error state gracefully', async () => {
    mockUseShipments.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Failed to load shipments'),
      refetch: vi.fn(),
    } as any);

    renderWithProviders(<ShipmentList />);

    await waitFor(() => {
      expect(screen.getByText('Error loading shipments')).toBeInTheDocument();
    });

    expect(screen.getByText('Failed to load shipments. Please try again.')).toBeInTheDocument();
  });

  it('has working navigation links', async () => {
    mockUseShipments.mockReturnValue({
      data: mockShipments,
      isLoading: false,
      error: null,
      refetch: vi.fn(),
    } as any);

    renderWithProviders(<ShipmentList />);

    await waitFor(() => {
      expect(screen.getByText('Test Package')).toBeInTheDocument();
    });

    // Check Add Shipment link
    const addLink = screen.getByRole('link', { name: /add shipment/i });
    expect(addLink).toHaveAttribute('href', '/shipments/new');

    // Check View links for shipments
    const viewLinks = screen.getAllByRole('link', { name: /view/i });
    expect(viewLinks[0]).toHaveAttribute('href', '/shipments/1');
  });

  it('has working refresh functionality', async () => {
    const user = userEvent.setup();
    const mockRefetch = vi.fn();
    
    mockUseShipments.mockReturnValue({
      data: mockShipments,
      isLoading: false,
      error: null,
      refetch: mockRefetch,
    } as any);

    renderWithProviders(<ShipmentList />);

    await waitFor(() => {
      expect(screen.getByText('Test Package')).toBeInTheDocument();
    });

    // Click refresh button
    const refreshButton = screen.getByRole('button', { name: /refresh/i });
    await user.click(refreshButton);

    expect(mockRefetch).toHaveBeenCalledOnce();
  });
});