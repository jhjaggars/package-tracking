import { Package, Truck, CheckCircle, AlertTriangle, Plus, Sparkles, Clock, MapPin } from 'lucide-react';
import { useDashboardStats, useShipments } from '../hooks/api';
import { Button } from '../components/ui/button';
import { Link } from 'react-router-dom';
import { sanitizePlainText } from '../lib/sanitize';
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
    // Skip animation in test environment
    if (import.meta.env?.MODE === 'test') {
      setAnimatedValue(numericValue);
      return;
    }
    
    if (!loading && numericValue > 0) {
      let timer: number;
      let counter: number;
      
      timer = setTimeout(() => {
        let start = 0;
        const duration = 1000;
        const increment = numericValue / (duration / 16);
        
        counter = setInterval(() => {
          start += increment;
          if (start >= numericValue) {
            setAnimatedValue(numericValue);
            clearInterval(counter);
          } else {
            setAnimatedValue(Math.floor(start));
          }
        }, 16);
      }, delay);
      
      // Proper cleanup for both timer and counter
      return () => {
        clearTimeout(timer);
        if (counter) {
          clearInterval(counter);
        }
      };
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
    <div className="bg-card p-6 rounded-xl border shadow-sm hover:shadow-md transition-all duration-200 cursor-pointer group">
      <div className="flex items-center">
        <div className="flex-shrink-0">
          <div className={`p-3 rounded-lg ${getColorClasses(color)}`}>
            <Icon className="h-6 w-6" />
          </div>
        </div>
        <div className="ml-5 w-0 flex-1">
          <dl>
            <dt className="text-sm font-medium text-muted-foreground truncate group-hover:text-foreground transition-colors">
              {title}
            </dt>
            <dd className="text-3xl font-bold text-card-foreground">
              {loading ? (
                <div>
                  ••••
                </div>
              ) : typeof value === 'number' ? (
                <span>
                  {animatedValue}
                </span>
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
    </div>
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
      <div className="md:flex md:items-center md:justify-between">
        <div className="flex-1 min-w-0">
          <div>
            <h1 className="text-3xl font-bold leading-7 text-foreground sm:text-4xl bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
              {getGreeting()}
            </h1>
            <p className="mt-2 text-lg text-muted-foreground flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-yellow-500" />
              {getSmartInsight()}
            </p>
          </div>
        </div>
        <div className="mt-4 flex md:mt-0 md:ml-4">
          <div>
            <Button asChild className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white border-0 shadow-lg">
              <Link to="/shipments/new">
                <Plus className="mr-2 h-4 w-4" />
                Add Shipment
              </Link>
            </Button>
          </div>
        </div>
      </div>

      {/* Delightful Stats Grid */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
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
      </div>

      {/* Delightful Recent Shipments */}
      <div className="bg-card shadow-lg rounded-xl border-0">
        <div className="px-6 py-6 sm:p-8">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-xl leading-6 font-semibold text-card-foreground flex items-center gap-2">
              <Clock className="h-5 w-5 text-blue-600" />
              Recent Activity
            </h3>
            <div>
              <Button variant="outline" size="sm" asChild className="hover:bg-blue-50 hover:border-blue-300">
                <Link to="/shipments">View All</Link>
              </Button>
            </div>
          </div>
          
            {shipmentsLoading ? (
              <div className="text-center py-8">
                <div className="mx-auto w-8 h-8 border-2 border-blue-600 border-t-transparent rounded-full animate-spin" />
                <p className="mt-4 text-muted-foreground">Loading your shipments...</p>
              </div>
            ) : recentShipments.length === 0 ? (
              <div className="text-center py-12">
                <div>
                  <Package className="mx-auto h-16 w-16 text-blue-400" />
                </div>
                <h3 className="mt-4 text-lg font-medium text-foreground">
                  Ready to track your first package?
                </h3>
                <p className="mt-2 text-muted-foreground max-w-sm mx-auto">
                  Add your tracking number and watch the magic happen as we keep you updated on every step of your delivery journey.
                </p>
                <div className="mt-8">
                  <Button asChild className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white border-0 shadow-lg">
                    <Link to="/shipments/new">
                      <Plus className="mr-2 h-4 w-4" />
                      Add Your First Shipment
                    </Link>
                  </Button>
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                {recentShipments.map((shipment) => (
                  <div
                    key={shipment.id}
                    className="flex items-center justify-between p-4 border rounded-xl hover:bg-gradient-to-r hover:from-blue-50 hover:to-purple-50 transition-all duration-200 cursor-pointer group"
                  >
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center space-x-4">
                        <div className="flex-shrink-0">
                          <div 
                            className={`w-3 h-3 rounded-full ${
                              shipment.is_delivered 
                                ? 'bg-green-500 shadow-green-200' 
                                : shipment.status === 'exception' 
                                ? 'bg-red-500 shadow-red-200'
                                : 'bg-blue-500 shadow-blue-200'
                            } shadow-lg`}
                          />
                        </div>
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
                              <span className="text-xs bg-green-100 text-green-800 px-2 py-1 rounded-full font-medium flex items-center gap-1">
                                <CheckCircle className="h-3 w-3" />
                                Delivered
                              </span>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                    <div className="flex-shrink-0">
                      <Button variant="ghost" size="sm" asChild className="hover:bg-blue-100 hover:text-blue-700">
                        <Link to={`/shipments/${shipment.id}`}>
                          <MapPin className="mr-1 h-3 w-3" />
                          Track
                        </Link>
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            )}
        </div>
      </div>
    </div>
  );
}