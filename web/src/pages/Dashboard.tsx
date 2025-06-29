import { Package, Truck, CheckCircle, AlertTriangle, Plus, Sparkles, Clock, MapPin } from 'lucide-react';
import { useDashboardStats, useShipments } from '../hooks/api';
import { Button } from '../components/ui/button';
import { Link } from 'react-router-dom';
import { sanitizePlainText } from '../lib/sanitize';
import { motion, AnimatePresence } from 'framer-motion';
import { useState, useEffect } from 'react';
import confetti from 'canvas-confetti';

function StatCard({ 
  title, 
  value, 
  icon: Icon, 
  description,
  loading = false,
  color = 'primary',
  delay = 0 
}: {
  title: string;
  value: number | string;
  icon: React.ElementType;
  description?: string;
  loading?: boolean;
  color?: string;
  delay?: number;
}) {
  const [animatedValue, setAnimatedValue] = useState(0);
  const numericValue = typeof value === 'number' ? value : 0;

  useEffect(() => {
    if (!loading && numericValue > 0) {
      const timer = setTimeout(() => {
        let start = 0;
        const duration = 1000;
        const increment = numericValue / (duration / 16);
        
        const counter = setInterval(() => {
          start += increment;
          if (start >= numericValue) {
            setAnimatedValue(numericValue);
            clearInterval(counter);
          } else {
            setAnimatedValue(Math.floor(start));
          }
        }, 16);
        
        return () => clearInterval(counter);
      }, delay);
      
      return () => clearTimeout(timer);
    }
  }, [loading, numericValue, delay]);

  const getColorClasses = (colorName: string) => {
    const colors = {
      primary: 'text-blue-600 bg-blue-50 border-blue-200',
      success: 'text-green-600 bg-green-50 border-green-200',
      warning: 'text-amber-600 bg-amber-50 border-amber-200',
      danger: 'text-red-600 bg-red-50 border-red-200'
    };
    return colors[colorName as keyof typeof colors] || colors.primary;
  };

  return (
    <motion.div 
      initial={{ opacity: 0, y: 20, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      transition={{ 
        duration: 0.5, 
        delay: delay * 0.1,
        type: "spring",
        stiffness: 100,
        damping: 12
      }}
      whileHover={{ 
        scale: 1.02, 
        transition: { duration: 0.2 } 
      }}
      className="bg-card p-6 rounded-xl border shadow-sm hover:shadow-md transition-all duration-200 cursor-pointer group"
    >
      <div className="flex items-center">
        <motion.div 
          className="flex-shrink-0"
          whileHover={{ rotate: 5 }}
          transition={{ type: "spring", stiffness: 300 }}
        >
          <div className={`p-3 rounded-lg ${getColorClasses(color)}`}>
            <Icon className="h-6 w-6" />
          </div>
        </motion.div>
        <div className="ml-5 w-0 flex-1">
          <dl>
            <dt className="text-sm font-medium text-muted-foreground truncate group-hover:text-foreground transition-colors">
              {title}
            </dt>
            <dd className="text-3xl font-bold text-card-foreground">
              {loading ? (
                <motion.div
                  animate={{ opacity: [1, 0.5, 1] }}
                  transition={{ duration: 1.5, repeat: Infinity }}
                >
                  ••••
                </motion.div>
              ) : typeof value === 'number' ? (
                <motion.span
                  key={animatedValue}
                  initial={{ scale: 1.2, opacity: 0.8 }}
                  animate={{ scale: 1, opacity: 1 }}
                  transition={{ duration: 0.3 }}
                >
                  {animatedValue}
                </motion.span>
              ) : value}
            </dd>
            {description && (
              <dd className="text-sm text-muted-foreground">
                {description}
              </dd>
            )}
          </dl>
        </div>
      </div>
    </motion.div>
  );
}

export function Dashboard() {
  const { data: stats, isLoading: statsLoading } = useDashboardStats();
  const { data: shipments, isLoading: shipmentsLoading } = useShipments();
  const [showConfetti, setShowConfetti] = useState(false);

  // Get recent shipments (last 5)
  const recentShipments = shipments?.slice(0, 5) || [];
  const deliveredToday = shipments?.filter(s => {
    const today = new Date().toDateString();
    return s.is_delivered && s.updated_at && new Date(s.updated_at).toDateString() === today;
  }) || [];

  // Greeting based on time of day
  const getGreeting = () => {
    const hour = new Date().getHours();
    if (hour < 12) return 'Good morning! ☕';
    if (hour < 17) return 'Good afternoon! ☀️';
    return 'Good evening! 🌙';
  };

  // Trigger confetti for deliveries
  useEffect(() => {
    if (deliveredToday.length > 0 && !showConfetti) {
      setShowConfetti(true);
      confetti({
        particleCount: 100,
        spread: 70,
        origin: { y: 0.6 },
        colors: ['#10B981', '#3B82F6', '#8B5CF6']
      });
    }
  }, [deliveredToday.length, showConfetti]);

  // Smart insights
  const getSmartInsight = () => {
    if (deliveredToday.length > 0) {
      return `🎉 ${deliveredToday.length} package${deliveredToday.length > 1 ? 's' : ''} delivered today!`;
    }
    const inTransit = stats?.in_transit || 0;
    if (inTransit > 0) {
      return `🚚 ${inTransit} package${inTransit > 1 ? 's are' : ' is'} on the way to you`;
    }
    return '📦 Your packages are all accounted for';
  };

  return (
    <div className="space-y-6">
      {/* Delightful Header */}
      <motion.div 
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6 }}
        className="md:flex md:items-center md:justify-between"
      >
        <div className="flex-1 min-w-0">
          <motion.div
            initial={{ opacity: 0, x: -20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ delay: 0.2, duration: 0.5 }}
          >
            <h1 className="text-3xl font-bold leading-7 text-foreground sm:text-4xl bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
              {getGreeting()}
            </h1>
            <motion.p 
              className="mt-2 text-lg text-muted-foreground flex items-center gap-2"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.4 }}
            >
              <Sparkles className="h-5 w-5 text-yellow-500" />
              {getSmartInsight()}
            </motion.p>
          </motion.div>
        </div>
        <motion.div 
          className="mt-4 flex md:mt-0 md:ml-4"
          initial={{ opacity: 0, scale: 0.9 }}
          animate={{ opacity: 1, scale: 1 }}
          transition={{ delay: 0.6 }}
        >
          <motion.div
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
          >
            <Button asChild className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white border-0 shadow-lg">
              <Link to="/shipments/new">
                <Plus className="mr-2 h-4 w-4" />
                Add Shipment
              </Link>
            </Button>
          </motion.div>
        </motion.div>
      </motion.div>

      {/* Delightful Stats Grid */}
      <motion.div 
        className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.8, staggerChildren: 0.1 }}
      >
        <StatCard
          title="Total Shipments"
          value={stats?.total_shipments || 0}
          icon={Package}
          loading={statsLoading}
          color="primary"
          delay={0}
        />
        <StatCard
          title="In Transit"
          value={stats?.in_transit || 0}
          icon={Truck}
          description="Currently shipping"
          loading={statsLoading}
          color="primary"
          delay={1}
        />
        <StatCard
          title="Delivered"
          value={stats?.delivered || 0}
          icon={CheckCircle}
          description="Successfully delivered"
          loading={statsLoading}
          color="success"
          delay={2}
        />
        <StatCard
          title="Requiring Attention"
          value={stats?.requiring_attention || 0}
          icon={AlertTriangle}
          description="Issues or exceptions"
          loading={statsLoading}
          color="warning"
          delay={3}
        />
      </motion.div>

      {/* Delightful Recent Shipments */}
      <motion.div 
        className="bg-card shadow-lg rounded-xl border-0"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 1.2, duration: 0.5 }}
      >
        <div className="px-6 py-6 sm:p-8">
          <motion.div 
            className="flex items-center justify-between mb-6"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 1.4 }}
          >
            <h3 className="text-xl leading-6 font-semibold text-card-foreground flex items-center gap-2">
              <Clock className="h-5 w-5 text-blue-600" />
              Recent Activity
            </h3>
            <motion.div
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <Button variant="outline" size="sm" asChild className="hover:bg-blue-50 hover:border-blue-300">
                <Link to="/shipments">View All</Link>
              </Button>
            </motion.div>
          </motion.div>
          
          <AnimatePresence>
            {shipmentsLoading ? (
              <motion.div 
                className="text-center py-8"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
              >
                <motion.div
                  animate={{ rotate: 360 }}
                  transition={{ duration: 2, repeat: Infinity, ease: "linear" }}
                  className="mx-auto w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full"
                />
                <p className="mt-4 text-muted-foreground">Loading your shipments...</p>
              </motion.div>
            ) : recentShipments.length === 0 ? (
              <motion.div 
                className="text-center py-12"
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: 0.2 }}
              >
                <motion.div
                  animate={{ 
                    y: [0, -10, 0],
                    rotate: [0, 5, -5, 0]
                  }}
                  transition={{ 
                    duration: 2, 
                    repeat: Infinity,
                    ease: "easeInOut"
                  }}
                >
                  <Package className="mx-auto h-16 w-16 text-blue-400" />
                </motion.div>
                <h3 className="mt-4 text-lg font-medium text-foreground">
                  Ready to track your first package?
                </h3>
                <p className="mt-2 text-muted-foreground max-w-sm mx-auto">
                  Add your tracking number and watch the magic happen as we keep you updated on every step of your delivery journey.
                </p>
                <motion.div 
                  className="mt-8"
                  whileHover={{ scale: 1.05 }}
                  whileTap={{ scale: 0.95 }}
                >
                  <Button asChild className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white border-0 shadow-lg">
                    <Link to="/shipments/new">
                      <Plus className="mr-2 h-4 w-4" />
                      Add Your First Shipment
                    </Link>
                  </Button>
                </motion.div>
              </motion.div>
            ) : (
              <motion.div 
                className="space-y-4"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                transition={{ delay: 0.3, staggerChildren: 0.1 }}
              >
                {recentShipments.map((shipment, index) => (
                  <motion.div
                    key={shipment.id}
                    initial={{ opacity: 0, x: -20 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.1 }}
                    whileHover={{ 
                      scale: 1.02,
                      boxShadow: "0 4px 12px rgba(0,0,0,0.1)"
                    }}
                    className="flex items-center justify-between p-4 border rounded-xl hover:bg-gradient-to-r hover:from-blue-50 hover:to-purple-50 transition-all duration-200 cursor-pointer group"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center space-x-4">
                        <motion.div 
                          className="flex-shrink-0"
                          whileHover={{ scale: 1.2 }}
                        >
                          <motion.div 
                            className={`w-3 h-3 rounded-full ${
                              shipment.is_delivered 
                                ? 'bg-green-500 shadow-green-200' 
                                : shipment.status === 'exception' 
                                ? 'bg-red-500 shadow-red-200'
                                : 'bg-blue-500 shadow-blue-200'
                            } shadow-lg`}
                            animate={shipment.is_delivered ? {} : {
                              scale: [1, 1.2, 1],
                              opacity: [1, 0.7, 1]
                            }}
                            transition={{
                              duration: 2,
                              repeat: Infinity,
                              ease: "easeInOut"
                            }}
                          />
                        </motion.div>
                        <div className="flex-1 min-w-0">
                          <p className="text-sm font-semibold text-foreground truncate group-hover:text-blue-700 transition-colors">
                            {sanitizePlainText(shipment.description)}
                          </p>
                          <div className="flex items-center gap-2 mt-1">
                            <p className="text-sm text-muted-foreground">
                              {shipment.tracking_number}
                            </p>
                            <span className="text-muted-foreground">•</span>
                            <span className={`text-xs font-medium px-2 py-1 rounded-full ${
                              shipment.carrier === 'ups' ? 'bg-amber-100 text-amber-800' :
                              shipment.carrier === 'fedex' ? 'bg-purple-100 text-purple-800' :
                              shipment.carrier === 'usps' ? 'bg-blue-100 text-blue-800' :
                              'bg-gray-100 text-gray-800'
                            }`}>
                              {shipment.carrier.toUpperCase()}
                            </span>
                            {shipment.is_delivered && (
                              <motion.span
                                initial={{ scale: 0 }}
                                animate={{ scale: 1 }}
                                className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded-full font-medium flex items-center gap-1"
                              >
                                <CheckCircle className="h-3 w-3" />
                                Delivered
                              </motion.span>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                    <motion.div 
                      className="flex-shrink-0"
                      whileHover={{ scale: 1.1 }}
                    >
                      <Button variant="ghost" size="sm" asChild className="hover:bg-blue-100 hover:text-blue-700">
                        <Link to={`/shipments/${shipment.id}`}>
                          <MapPin className="mr-1 h-3 w-3" />
                          Track
                        </Link>
                      </Button>
                    </motion.div>
                  </motion.div>
                ))}
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </motion.div>
    </div>
  );
}