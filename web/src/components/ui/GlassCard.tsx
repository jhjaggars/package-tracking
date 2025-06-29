import { motion } from 'framer-motion';
import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';

interface GlassCardProps {
  children: ReactNode;
  className?: string;
  blur?: 'sm' | 'md' | 'lg' | 'xl';
  opacity?: 'low' | 'medium' | 'high';
  glow?: boolean;
  hover3d?: boolean;
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
  hover3d = true
}: GlassCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      whileHover={hover3d ? { 
        y: -8, 
        rotateX: 5,
        rotateY: 5,
        scale: 1.02
      } : { y: -2 }}
      transition={{ 
        type: "spring", 
        stiffness: 300, 
        damping: 30 
      }}
      className={cn(
        'relative group',
        'rounded-3xl border border-white/20 dark:border-white/10',
        'p-6 overflow-hidden',
        blurLevels[blur],
        opacityLevels[opacity],
        'shadow-xl hover:shadow-2xl transition-all duration-500',
        glow && 'hover:shadow-blue-500/25 dark:hover:shadow-blue-400/20',
        className
      )}
      style={{
        transformStyle: 'preserve-3d',
      }}
    >
      {/* Animated gradient border */}
      <div className="absolute inset-0 rounded-3xl bg-gradient-to-r from-blue-500/20 via-purple-500/20 to-pink-500/20 opacity-0 group-hover:opacity-100 transition-opacity duration-500 blur-sm" />
      
      {/* Content container */}
      <div className="relative z-10">
        {children}
      </div>
      
      {/* Floating orbs for extra visual interest */}
      <div className="absolute -top-4 -right-4 w-8 h-8 bg-gradient-to-br from-blue-400 to-purple-500 rounded-full opacity-20 group-hover:opacity-40 transition-opacity duration-500 blur-sm" />
      <div className="absolute -bottom-4 -left-4 w-6 h-6 bg-gradient-to-br from-pink-400 to-orange-500 rounded-full opacity-15 group-hover:opacity-30 transition-opacity duration-500 blur-sm" />
    </motion.div>
  );
}