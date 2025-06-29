import { motion } from 'framer-motion';

interface ModernLoaderProps {
  type?: 'neural' | 'particles' | 'morphing' | 'wave';
  size?: 'sm' | 'md' | 'lg';
  color?: 'blue' | 'purple' | 'green' | 'multicolor';
}

const sizes = {
  sm: 'w-8 h-8',
  md: 'w-12 h-12',
  lg: 'w-16 h-16',
};

const colors = {
  blue: 'text-blue-600',
  purple: 'text-purple-600',
  green: 'text-green-600',
  multicolor: 'text-blue-600',
};

function NeuralLoader({ size, color }: { size: string; color: string }) {
  const dots = Array.from({ length: 9 }, (_, i) => i);
  
  return (
    <div className={`${size} relative flex items-center justify-center`}>
      <div className="grid grid-cols-3 gap-1">
        {dots.map((i) => (
          <motion.div
            key={i}
            className={`w-1.5 h-1.5 bg-current rounded-full ${color}`}
            animate={{
              scale: [1, 1.5, 1],
              opacity: [0.3, 1, 0.3],
            }}
            transition={{
              duration: 1.5,
              repeat: Infinity,
              delay: i * 0.15,
              ease: "easeInOut",
            }}
          />
        ))}
      </div>
      
      {/* Neural connections */}
      {[0, 1, 2].map((row) => (
        [0, 1].map((col) => (
          <motion.div
            key={`${row}-${col}`}
            className={`absolute w-px h-3 bg-current ${color} opacity-20`}
            style={{
              left: `${30 + col * 20}%`,
              top: `${25 + row * 25}%`,
              transformOrigin: 'center',
            }}
            animate={{
              scaleY: [0, 1, 0],
              opacity: [0, 0.5, 0],
            }}
            transition={{
              duration: 2,
              repeat: Infinity,
              delay: row * 0.3 + col * 0.1,
            }}
          />
        ))
      ))}
    </div>
  );
}

function ParticleLoader({ size, color }: { size: string; color: string }) {
  const particles = Array.from({ length: 12 }, (_, i) => i);
  
  return (
    <div className={`${size} relative`}>
      {particles.map((i) => {
        const angle = (i * 360) / particles.length;
        return (
          <motion.div
            key={i}
            className={`absolute w-2 h-2 bg-current rounded-full ${color}`}
            style={{
              left: '50%',
              top: '50%',
            }}
            animate={{
              x: [0, Math.cos(angle * Math.PI / 180) * 20],
              y: [0, Math.sin(angle * Math.PI / 180) * 20],
              scale: [0, 1, 0],
              opacity: [0, 1, 0],
            }}
            transition={{
              duration: 2,
              repeat: Infinity,
              delay: i * 0.1,
              ease: "easeInOut",
            }}
          />
        );
      })}
    </div>
  );
}

function MorphingLoader({ size, color }: { size: string; color: string }) {
  return (
    <div className={`${size} flex items-center justify-center`}>
      <motion.div
        className={`w-full h-full bg-current ${color} rounded-full`}
        animate={{
          borderRadius: [
            '50%',
            '25% 75% 45% 55%',
            '75% 25% 55% 45%',
            '50%',
          ],
          rotate: [0, 180, 360],
        }}
        transition={{
          duration: 3,
          repeat: Infinity,
          ease: "easeInOut",
        }}
      />
    </div>
  );
}

function WaveLoader({ size, color }: { size: string; color: string }) {
  const bars = Array.from({ length: 5 }, (_, i) => i);
  
  return (
    <div className={`${size} flex items-center justify-center gap-1`}>
      {bars.map((i) => (
        <motion.div
          key={i}
          className={`w-1 bg-current ${color} rounded-full`}
          animate={{
            height: ['20%', '100%', '20%'],
            opacity: [0.3, 1, 0.3],
          }}
          transition={{
            duration: 1.2,
            repeat: Infinity,
            delay: i * 0.15,
            ease: "easeInOut",
          }}
        />
      ))}
    </div>
  );
}

export function ModernLoader({ 
  type = 'neural', 
  size = 'md', 
  color = 'blue' 
}: ModernLoaderProps) {
  const sizeClass = sizes[size];
  const colorClass = colors[color];
  
  const loaderComponents = {
    neural: <NeuralLoader size={sizeClass} color={colorClass} />,
    particles: <ParticleLoader size={sizeClass} color={colorClass} />,
    morphing: <MorphingLoader size={sizeClass} color={colorClass} />,
    wave: <WaveLoader size={sizeClass} color={colorClass} />,
  };
  
  return (
    <div className="flex flex-col items-center justify-center space-y-4">
      {loaderComponents[type]}
      <motion.p
        animate={{ opacity: [0.5, 1, 0.5] }}
        transition={{ duration: 2, repeat: Infinity }}
        className="text-sm text-muted-foreground"
      >
        Loading your packages...
      </motion.p>
    </div>
  );
}