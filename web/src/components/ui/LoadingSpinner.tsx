import { Package } from 'lucide-react';

export function LoadingSpinner() {
  return (
    <div className="flex flex-col items-center justify-center min-h-[200px] space-y-4">
      <div className="relative">
        <div className="w-12 h-12 border-4 border-blue-200 border-t-blue-600 rounded-full animate-spin"></div>
        <div className="absolute inset-2 flex items-center justify-center">
          <Package className="h-4 w-4 text-blue-600" />
        </div>
      </div>
      <p className="text-sm text-muted-foreground">
        Loading your packages...
      </p>
    </div>
  );
}