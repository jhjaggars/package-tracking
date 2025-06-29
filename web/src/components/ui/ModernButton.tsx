import React from 'react';
import { cn } from '../../lib/utils';

interface ModernButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: 'primary' | 'secondary' | 'ghost' | 'gradient';
  size?: 'sm' | 'md' | 'lg';
  loading?: boolean;
  icon?: React.ReactNode;
}

const variants = {
  primary: 'bg-[var(--color-primary)] hover:bg-[var(--color-primary-dark)] text-white',
  secondary: 'bg-[var(--bg-secondary)] hover:bg-[var(--bg-tertiary)] text-[var(--text-primary)]',
  ghost: 'bg-transparent hover:bg-[var(--bg-secondary)] text-[var(--text-primary)]',
  gradient: 'bg-gradient-primary text-white shadow-lg hover:shadow-xl',
};

const sizes = {
  sm: 'px-3 py-1.5 text-sm',
  md: 'px-4 py-2 text-base',
  lg: 'px-6 py-3 text-lg',
};

export const ModernButton: React.FC<ModernButtonProps> = ({
  children,
  className,
  variant = 'primary',
  size = 'md',
  loading = false,
  icon,
  disabled,
  ...props
}) => {
  return (
    <button
      className={cn(
        'relative overflow-hidden',
        'font-medium rounded-lg',
        'focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-[var(--color-primary)]',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        variants[variant],
        sizes[size],
        loading && 'cursor-wait',
        className
      )}
      disabled={disabled || loading}
      {...props}
    >
      <span className={cn(
        'flex items-center justify-center gap-2',
        loading && 'opacity-0'
      )}>
        {icon && <span className="w-5 h-5">{icon}</span>}
        {children}
      </span>
      
      {loading && (
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
        </div>
      )}
      
      {/* Ripple effect removed */}
    </button>
  );
};