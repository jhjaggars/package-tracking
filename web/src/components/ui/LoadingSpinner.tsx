import { motion } from 'framer-motion';
import { Package } from 'lucide-react';

export function LoadingSpinner() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[200px] space-y-4">
      <motion.div
        animate={{ 
          rotate: 360,
          scale: [1, 1.1, 1]
        }}
        transition={{ 
          rotate: { duration: 2, repeat: Infinity, ease: "linear" },
          scale: { duration: 1, repeat: Infinity, ease: "easeInOut" }
        }}
        className="relative"
      >
        <div className="w-12 h-12 border-4 border-blue-200 border-t-blue-600 rounded-full"></div>
        <motion.div
          animate={{ rotate: -360 }}
          transition={{ duration: 3, repeat: Infinity, ease: "linear" }}
          className="absolute inset-2 flex items-center justify-center"
        >
          <Package className="h-4 w-4 text-blue-600" />
        </motion.div>
      </motion.div>
      <motion.p
        animate={{ opacity: [1, 0.5, 1] }}
        transition={{ duration: 1.5, repeat: Infinity }}
        className="text-sm text-muted-foreground"
      >
        Loading your packages...
      </motion.p>
    </div>
  );
}