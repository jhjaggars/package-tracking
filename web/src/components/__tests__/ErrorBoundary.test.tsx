import { describe, it, expect, vi, beforeEach } from 'vitest';
import { screen } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { renderWithProviders } from '../../test/utils';
import { ErrorBoundary } from '../ErrorBoundary';

// Component that throws an error for testing
function ThrowingComponent({ shouldThrow = false }: { shouldThrow?: boolean }) {
  if (shouldThrow) {
    throw new Error('Test error message');
  }
  return <div>Working component</div>;
}

// Mock import.meta.env
const originalEnv = import.meta.env;
Object.defineProperty(import.meta, 'env', {
  value: {
    ...originalEnv,
    DEV: false,
  },
  writable: true,
});

describe('ErrorBoundary', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset to production mode by default
    (import.meta.env as any).DEV = false;
  });

  it('renders children when there is no error', () => {
    renderWithProviders(
      <ErrorBoundary>
        <ThrowingComponent shouldThrow={false} />
      </ErrorBoundary>
    );

    expect(screen.getByText('Working component')).toBeInTheDocument();
  });

  it('catches errors and displays error UI', () => {
    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    renderWithProviders(
      <ErrorBoundary>
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText(/An unexpected error occurred/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /try again/i })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /refresh page/i })).toBeInTheDocument();

    consoleSpy.mockRestore();
  });

  it('displays error information', () => {
    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    renderWithProviders(
      <ErrorBoundary>
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
    expect(screen.getByText(/An unexpected error occurred/)).toBeInTheDocument();

    consoleSpy.mockRestore();
  });

  it('provides try again button', async () => {
    const user = userEvent.setup();
    
    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    renderWithProviders(
      <ErrorBoundary>
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    );

    // Error boundary should be showing
    expect(screen.getByText('Something went wrong')).toBeInTheDocument();

    // Should have "Try Again" button
    const tryAgainButton = screen.getByRole('button', { name: /try again/i });
    expect(tryAgainButton).toBeInTheDocument();
    
    // Clicking it should be possible (testing interaction)
    await user.click(tryAgainButton);

    consoleSpy.mockRestore();
  });

  it('handles refresh page button', async () => {
    const user = userEvent.setup();
    
    // Mock window.location.reload
    const mockReload = vi.fn();
    Object.defineProperty(window, 'location', {
      value: { reload: mockReload },
      writable: true,
    });

    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    renderWithProviders(
      <ErrorBoundary>
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    );

    const refreshButton = screen.getByRole('button', { name: /refresh page/i });
    await user.click(refreshButton);

    expect(mockReload).toHaveBeenCalledOnce();

    consoleSpy.mockRestore();
  });

  it('renders custom fallback when provided', () => {
    // Suppress console.error for this test
    const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

    const customFallback = <div>Custom error message</div>;

    renderWithProviders(
      <ErrorBoundary fallback={customFallback}>
        <ThrowingComponent shouldThrow={true} />
      </ErrorBoundary>
    );

    expect(screen.getByText('Custom error message')).toBeInTheDocument();
    expect(screen.queryByText('Something went wrong')).not.toBeInTheDocument();

    consoleSpy.mockRestore();
  });
});