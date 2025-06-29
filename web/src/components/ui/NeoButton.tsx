import { motion } from 'framer-motion';
import type { ReactNode } from 'react';
import { cn } from '../../lib/utils';

interface NeoButtonProps {
  children: ReactNode;
  onClick?: () => void;
  variant?: 'primary' | 'secondary' | 'success' | 'warning';
  size?: 'sm' | 'md' | 'lg';
  className?: string;
  disabled?: boolean;
}

const variants = {
  primary: `
    bg-gradient-to-br from-blue-400 to-blue-600 text-white
    shadow-[8px_8px_16px_#dde4f0,_-8px_-8px_16px_#ffffff]
    dark:shadow-[8px_8px_16px_#0f172a,_-8px_-8px_16px_#1e293b]
    hover:shadow-[4px_4px_8px_#dde4f0,_-4px_-4px_8px_#ffffff]
    dark:hover:shadow-[4px_4px_8px_#0f172a,_-4px_-4px_8px_#1e293b]
    active:shadow-[inset_4px_4px_8px_#dde4f0,_inset_-4px_-4px_8px_#ffffff]
    dark:active:shadow-[inset_4px_4px_8px_#0f172a,_inset_-4px_-4px_8px_#1e293b]
  `,
  secondary: `
    bg-gray-100 dark:bg-gray-800 text-gray-800 dark:text-gray-200
    shadow-[8px_8px_16px_#d1d9e6,_-8px_-8px_16px_#ffffff]
    dark:shadow-[8px_8px_16px_#0f172a,_-8px_-8px_16px_#2d3748]
    hover:shadow-[4px_4px_8px_#d1d9e6,_-4px_-4px_8px_#ffffff]
    dark:hover:shadow-[4px_4px_8px_#0f172a,_-4px_-4px_8px_#2d3748]
  `,
  success: `
    bg-gradient-to-br from-green-400 to-green-600 text-white
    shadow-[8px_8px_16px_#d1f2d9,_-8px_-8px_16px_#ffffff]
    dark:shadow-[8px_8px_16px_#0f2d15,_-8px_-8px_16px_#1a4a22]
  `,
  warning: `
    bg-gradient-to-br from-amber-400 to-amber-600 text-white
    shadow-[8px_8px_16px_#fef3cd,_-8px_-8px_16px_#ffffff]
    dark:shadow-[8px_8px_16px_#2d1b00,_-8px_-8px_16px_#4a2f00]
  `,
};

const sizes = {
  sm: 'px-4 py-2 text-sm rounded-xl',
  md: 'px-6 py-3 text-base rounded-2xl',
  lg: 'px-8 py-4 text-lg rounded-3xl',
};

export function NeoButton({ 
  children, 
  onClick, 
  variant = 'primary', 
  size = 'md',
  className = '',
  disabled = false 
}: NeoButtonProps) {
  return (
    <motion.button
      onClick={onClick}
      disabled={disabled}
      whileTap={{ scale: 0.98 }}
      whileHover={{ scale: 1.02 }}
      transition={{ type: "spring", stiffness: 400, damping: 10 }}
      className={cn(
        'relative overflow-hidden border-0 font-medium transition-all duration-300',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        variants[variant],
        sizes[size],
        className
      )}
    >
      {/* Shine effect */}
      <div className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent -skew-x-12 -translate-x-full group-hover:translate-x-full transition-transform duration-1000" />
      
      <span className="relative z-10 flex items-center justify-center gap-2">
        {children}
      </span>
    </motion.button>
  );
}