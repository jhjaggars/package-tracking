import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';

interface GlassCardProps {
  children: ReactNode;
  className?: string;
  blur?: 'sm' | 'md' | 'lg' | 'xl';
  opacity?: 'low' | 'medium' | 'high';
  glow?: boolean;
}

const blurLevels = {
  sm: 'backdrop-blur-sm',
  md: 'backdrop-blur-md',
  lg: 'backdrop-blur-lg',
  xl: 'backdrop-blur-xl',
};

const opacityLevels = {
  low: 'bg-white/10 dark:bg-white/5',
  medium: 'bg-white/20 dark:bg-white/10',
  high: 'bg-white/30 dark:bg-white/15',
};

export function GlassCard({ 
  children, 
  className = '',
  blur = 'md',
  opacity = 'medium',
  glow = false,
}: GlassCardProps) {
  return (
    <div
      className={cn(
        'relative group',
        'rounded-3xl border border-white/20 dark:border-white/10',
        'p-6 overflow-hidden',
        blurLevels[blur],
        opacityLevels[opacity],
        'shadow-xl hover:shadow-2xl',
        glow && 'hover:shadow-blue-500/25 dark:hover:shadow-blue-400/20',
        className
      )}
    >
      {/* Static gradient border */}
      <div className="absolute inset-0 rounded-3xl bg-gradient-to-r from-blue-500/20 via-purple-500/20 to-pink-500/20 opacity-0 group-hover:opacity-100 blur-sm" />
      
      {/* Content container */}
      <div className="relative z-10">
        {children}
      </div>
      
      {/* Static orbs for visual interest */}
      <div className="absolute -top-4 -right-4 w-8 h-8 bg-gradient-to-br from-blue-400 to-purple-500 rounded-full opacity-20 group-hover:opacity-40 blur-sm" />
      <div className="absolute -bottom-4 -left-4 w-6 h-6 bg-gradient-to-br from-pink-400 to-orange-500 rounded-full opacity-15 group-hover:opacity-30 blur-sm" />
    </div>
  );
}