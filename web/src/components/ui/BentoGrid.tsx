import type { ReactNode } from 'react';

interface BentoCardProps {
  children: ReactNode;
  className?: string;
  size?: 'small' | 'medium' | 'large';
  gradient?: 'blue' | 'purple' | 'green' | 'orange';
}

const sizeClasses = {
  small: 'col-span-1 row-span-1',
  medium: 'col-span-2 row-span-1 sm:col-span-1',
  large: 'col-span-2 row-span-2',
};

const gradientClasses = {
  blue: 'bg-gradient-to-br from-blue-50 to-blue-100 dark:from-blue-950 dark:to-blue-900 border-blue-200 dark:border-blue-800',
  purple: 'bg-gradient-to-br from-purple-50 to-purple-100 dark:from-purple-950 dark:to-purple-900 border-purple-200 dark:border-purple-800',
  green: 'bg-gradient-to-br from-green-50 to-green-100 dark:from-green-950 dark:to-green-900 border-green-200 dark:border-green-800',
  orange: 'bg-gradient-to-br from-orange-50 to-orange-100 dark:from-orange-950 dark:to-orange-900 border-orange-200 dark:border-orange-800',
};

export function BentoCard({ 
  children, 
  className = '', 
  size = 'medium',
  gradient = 'blue'
}: BentoCardProps) {
  return (
    <div
      className={`
        ${sizeClasses[size]}
        ${gradientClasses[gradient]}
        relative overflow-hidden rounded-2xl border backdrop-blur-sm
        p-6 shadow-lg hover:shadow-xl
        ${className}
      `}
    >
      {/* Subtle texture overlay */}
      <div className="absolute inset-0 bg-gradient-to-br from-white/10 to-transparent pointer-events-none" />
      
      {/* Content */}
      <div className="relative z-10">
        {children}
      </div>
    </div>
  );
}

export function BentoGrid({ children, className = '' }: { children: ReactNode; className?: string }) {
  return (
    <div className={`grid grid-cols-2 sm:grid-cols-4 gap-4 auto-rows-[200px] ${className}`}>
      {children}
    </div>
  );
}