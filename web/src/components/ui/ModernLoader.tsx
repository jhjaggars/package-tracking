interface ModernLoaderProps {
  type?: 'spinner' | 'dots' | 'pulse';
  size?: 'sm' | 'md' | 'lg';
  color?: 'blue' | 'purple' | 'green';
}

const sizes = {
  sm: 'w-6 h-6',
  md: 'w-8 h-8',
  lg: 'w-12 h-12',
};

const colors = {
  blue: 'border-blue-600',
  purple: 'border-purple-600',
  green: 'border-green-600',
};

const dotColors = {
  blue: 'bg-blue-600',
  purple: 'bg-purple-600',
  green: 'bg-green-600',
};

function SpinnerLoader({ size, color }: { size: string; color: string }) {
  return (
    <div className={`${size} relative`}>
      <div className={`w-full h-full border-4 border-gray-200 border-t-4 ${color} rounded-full animate-spin`}></div>
    </div>
  );
}

function DotsLoader({ color }: { color: string }) {
  const dots = Array.from({ length: 3 }, (_, i) => i);
  
  return (
    <div className="flex space-x-1">
      {dots.map((i) => (
        <div
          key={i}
          className={`w-2 h-2 ${color} rounded-full animate-pulse`}
          style={{
            animationDelay: `${i * 0.2}s`,
            animationDuration: '1s'
          }}
        />
      ))}
    </div>
  );
}

function PulseLoader({ size, color }: { size: string; color: string }) {
  return (
    <div className={`${size} relative`}>
      <div className={`w-full h-full ${color.replace('border-', 'bg-')} rounded-full animate-pulse`}></div>
    </div>
  );
}

export function ModernLoader({ 
  type = 'spinner', 
  size = 'md', 
  color = 'blue' 
}: ModernLoaderProps) {
  const sizeClass = sizes[size];
  const colorClass = colors[color];
  const dotColorClass = dotColors[color];
  
  const loaderComponents = {
    spinner: <SpinnerLoader size={sizeClass} color={colorClass} />,
    dots: <DotsLoader color={dotColorClass} />,
    pulse: <PulseLoader size={sizeClass} color={colorClass} />,
  };
  
  return (
    <div className="flex flex-col items-center justify-center space-y-4">
      <div className="prefers-reduced-motion:hidden">
        {loaderComponents[type]}
      </div>
      <div className="hidden prefers-reduced-motion:block">
        <div className={`${sizeClass} ${dotColorClass} rounded-full`}></div>
      </div>
      <p className="text-sm text-muted-foreground">
        Loading your packages...
      </p>
    </div>
  );
}