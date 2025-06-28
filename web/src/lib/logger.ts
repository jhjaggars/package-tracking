// Production-safe logging utility
const isDevelopment = import.meta.env.DEV;

export const logger = {
  debug: (...args: unknown[]) => {
    if (isDevelopment) {
      console.debug(...args);
    }
  },
  
  log: (...args: unknown[]) => {
    if (isDevelopment) {
      console.log(...args);
    }
  },
  
  info: (...args: unknown[]) => {
    if (isDevelopment) {
      console.info(...args);
    }
  },
  
  warn: (...args: unknown[]) => {
    // Always log warnings, even in production
    console.warn(...args);
  },
  
  error: (...args: unknown[]) => {
    // Always log errors, even in production
    console.error(...args);
  },
};