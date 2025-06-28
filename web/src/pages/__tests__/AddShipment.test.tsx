import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { renderWithProviders, mockCarrier } from '../../test/utils';
import { AddShipment } from '../AddShipment';
import * as api from '../../hooks/api';

// Mock the API hooks
vi.mock('../../hooks/api', () => ({
  useCarriers: vi.fn(),
  useCreateShipment: vi.fn(),
}));

// Mock useNavigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const mockUseCarriers = vi.mocked(api.useCarriers);
const mockUseCreateShipment = vi.mocked(api.useCreateShipment);

describe('AddShipment', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockNavigate.mockClear();
  });

  it('renders the form correctly', () => {
    mockUseCarriers.mockReturnValue({
      data: [mockCarrier],
      isLoading: false,
      error: null,
    } as any);

    mockUseCreateShipment.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      error: null,
    } as any);

    renderWithProviders(<AddShipment />);

    expect(screen.getByText('Add New Shipment')).toBeInTheDocument();
    expect(screen.getByLabelText('Tracking Number')).toBeInTheDocument();
    expect(screen.getByLabelText('Carrier')).toBeInTheDocument();
    expect(screen.getByLabelText('Description')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /add shipment/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /cancel/i })).toBeInTheDocument();
  });

  it('validates required fields', async () => {
    const user = userEvent.setup();
    
    mockUseCarriers.mockReturnValue({
      data: [mockCarrier],
      isLoading: false,
      error: null,
    } as any);

    mockUseCreateShipment.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      error: null,
    } as any);

    renderWithProviders(<AddShipment />);

    // Try to submit without filling required fields
    const submitButton = screen.getByRole('button', { name: /add shipment/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText('Tracking number is required')).toBeInTheDocument();
    });

    expect(screen.getByText('Carrier is required')).toBeInTheDocument();
    expect(screen.getByText('Description is required')).toBeInTheDocument();
  });

  it('validates tracking number format', async () => {
    const user = userEvent.setup();
    
    mockUseCarriers.mockReturnValue({
      data: [mockCarrier],
      isLoading: false,
      error: null,
    } as any);

    mockUseCreateShipment.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      error: null,
    } as any);

    renderWithProviders(<AddShipment />);

    const trackingInput = screen.getByLabelText('Tracking Number');
    await user.type(trackingInput, '123'); // Too short
    
    const submitButton = screen.getByRole('button', { name: /add shipment/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(screen.getByText('Tracking number must be at least 5 characters')).toBeInTheDocument();
    });
  });

  it('submits form with valid data', async () => {
    const user = userEvent.setup();
    const mockMutate = vi.fn();
    
    mockUseCarriers.mockReturnValue({
      data: [mockCarrier],
      isLoading: false,
      error: null,
    } as any);

    mockUseCreateShipment.mockReturnValue({
      mutate: mockMutate,
      isPending: false,
      error: null,
    } as any);

    renderWithProviders(<AddShipment />);

    // Fill out the form
    await user.type(screen.getByLabelText('Tracking Number'), '1Z999AA1234567890');
    await user.selectOptions(screen.getByLabelText('Carrier'), 'ups');
    await user.type(screen.getByLabelText('Description'), 'Test Package');

    // Submit the form
    const submitButton = screen.getByRole('button', { name: /add shipment/i });
    await user.click(submitButton);

    await waitFor(() => {
      expect(mockMutate).toHaveBeenCalledWith({
        tracking_number: '1Z999AA1234567890',
        carrier: 'ups',
        description: 'Test Package',
      });
    });
  });

  it('shows loading state during submission', () => {
    mockUseCarriers.mockReturnValue({
      data: [mockCarrier],
      isLoading: false,
      error: null,
    } as any);

    mockUseCreateShipment.mockReturnValue({
      mutate: vi.fn(),
      isPending: true,
      error: null,
    } as any);

    renderWithProviders(<AddShipment />);

    const submitButton = screen.getByRole('button', { name: /adding/i });
    expect(submitButton).toBeDisabled();
  });

  it('displays error message on submission failure', () => {
    mockUseCarriers.mockReturnValue({
      data: [mockCarrier],
      isLoading: false,
      error: null,
    } as any);

    mockUseCreateShipment.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      error: new Error('Failed to create shipment'),
    } as any);

    renderWithProviders(<AddShipment />);

    expect(screen.getByText('Failed to create shipment')).toBeInTheDocument();
  });

  it('handles carriers loading state', () => {
    mockUseCarriers.mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    } as any);

    mockUseCreateShipment.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      error: null,
    } as any);

    renderWithProviders(<AddShipment />);

    expect(screen.getByText('Loading carriers...')).toBeInTheDocument();
  });

  it('handles carriers error state', () => {
    mockUseCarriers.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Failed to load carriers'),
    } as any);

    mockUseCreateShipment.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      error: null,
    } as any);

    renderWithProviders(<AddShipment />);

    expect(screen.getByText('Error loading carriers')).toBeInTheDocument();
    expect(screen.getByText('Failed to load carriers. Please refresh the page.')).toBeInTheDocument();
  });

  it('navigates back on cancel', async () => {
    const user = userEvent.setup();
    
    mockUseCarriers.mockReturnValue({
      data: [mockCarrier],
      isLoading: false,
      error: null,
    } as any);

    mockUseCreateShipment.mockReturnValue({
      mutate: vi.fn(),
      isPending: false,
      error: null,
    } as any);

    renderWithProviders(<AddShipment />);

    const cancelButton = screen.getByRole('button', { name: /cancel/i });
    await user.click(cancelButton);

    expect(mockNavigate).toHaveBeenCalledWith('/shipments');
  });
});