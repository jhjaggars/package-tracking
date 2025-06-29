import { describe, it, expect, vi } from 'vitest';
import { screen } from '@testing-library/react';
import { userEvent } from '@testing-library/user-event';
import { renderWithProviders } from '../../../test/utils';
import { ModernButton } from '../ModernButton';

describe('ModernButton', () => {
  it('renders children correctly', () => {
    renderWithProviders(
      <ModernButton>Click me</ModernButton>
    );

    expect(screen.getByRole('button', { name: 'Click me' })).toBeInTheDocument();
  });

  it('handles click events', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();

    renderWithProviders(
      <ModernButton onClick={handleClick}>Click me</ModernButton>
    );

    await user.click(screen.getByRole('button', { name: 'Click me' }));
    expect(handleClick).toHaveBeenCalledOnce();
  });

  it('applies primary variant by default', () => {
    renderWithProviders(
      <ModernButton>Primary Button</ModernButton>
    );

    const button = screen.getByRole('button');
    expect(button).toHaveClass('bg-[var(--color-primary)]');
  });

  it('applies different variants correctly', () => {
    const { rerender } = renderWithProviders(
      <ModernButton variant="secondary">Secondary</ModernButton>
    );

    let button = screen.getByRole('button');
    expect(button).toHaveClass('bg-[var(--bg-secondary)]');

    rerender(
      <ModernButton variant="ghost">Ghost</ModernButton>
    );

    button = screen.getByRole('button');
    expect(button).toHaveClass('bg-transparent');

    rerender(
      <ModernButton variant="gradient">Gradient</ModernButton>
    );

    button = screen.getByRole('button');
    expect(button).toHaveClass('bg-gradient-primary');
  });

  it('applies different sizes correctly', () => {
    const { rerender } = renderWithProviders(
      <ModernButton size="sm">Small</ModernButton>
    );

    let button = screen.getByRole('button');
    expect(button).toHaveClass('px-3', 'py-1.5', 'text-sm');

    rerender(
      <ModernButton size="md">Medium</ModernButton>
    );

    button = screen.getByRole('button');
    expect(button).toHaveClass('px-4', 'py-2', 'text-base');

    rerender(
      <ModernButton size="lg">Large</ModernButton>
    );

    button = screen.getByRole('button');
    expect(button).toHaveClass('px-6', 'py-3', 'text-lg');
  });

  it('shows loading state correctly', () => {
    renderWithProviders(
      <ModernButton loading>Loading Button</ModernButton>
    );

    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button).toHaveClass('cursor-wait');
    
    // Should show loading spinner (check for animate-spin class)
    const spinner = document.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
  });

  it('disables button when loading', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();

    renderWithProviders(
      <ModernButton loading onClick={handleClick}>Loading</ModernButton>
    );

    const button = screen.getByRole('button');
    expect(button).toBeDisabled();

    // Clicking should not trigger the handler
    await user.click(button);
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('renders with icon', () => {
    const TestIcon = () => <span data-testid="test-icon">🎯</span>;

    renderWithProviders(
      <ModernButton icon={<TestIcon />}>With Icon</ModernButton>
    );

    expect(screen.getByTestId('test-icon')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: /with icon/i })).toBeInTheDocument();
  });

  it('accepts custom className', () => {
    renderWithProviders(
      <ModernButton className="custom-class">Custom</ModernButton>
    );

    const button = screen.getByRole('button');
    expect(button).toHaveClass('custom-class');
  });

  it('forwards other button props', () => {
    renderWithProviders(
      <ModernButton type="submit" data-testid="submit-button">Submit</ModernButton>
    );

    const button = screen.getByTestId('submit-button');
    expect(button).toHaveAttribute('type', 'submit');
  });

  it('handles disabled state', async () => {
    const user = userEvent.setup();
    const handleClick = vi.fn();

    renderWithProviders(
      <ModernButton disabled onClick={handleClick}>Disabled</ModernButton>
    );

    const button = screen.getByRole('button');
    expect(button).toBeDisabled();
    expect(button).toHaveClass('disabled:opacity-50', 'disabled:cursor-not-allowed');

    // Clicking should not trigger the handler
    await user.click(button);
    expect(handleClick).not.toHaveBeenCalled();
  });

  it('applies focus styles for accessibility', async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <ModernButton>Focus Test</ModernButton>
    );

    const button = screen.getByRole('button');
    expect(button).toHaveClass('focus:outline-none', 'focus:ring-2', 'focus:ring-offset-2');

    // Test that focus works
    await user.tab();
    expect(button).toHaveFocus();
  });

  it('hides content when loading but keeps accessibility', () => {
    renderWithProviders(
      <ModernButton loading>Loading Content</ModernButton>
    );

    // Content should be hidden (opacity-0) but still in DOM for screen readers
    const content = screen.getByText('Loading Content');
    const contentSpan = content.closest('span');
    expect(contentSpan).toHaveClass('opacity-0');
  });
});