import { Package, Truck, CheckCircle, AlertTriangle, Plus, Sparkles, TrendingUp, Clock } from 'lucide-react';
import { useDashboardStats, useShipments } from '../hooks/api';
import { Link } from 'react-router-dom';

// Modern components
import { BentoCard, BentoGrid } from '../components/ui/BentoGrid';
import { GlassCard } from '../components/ui/GlassCard';
import { NeoButton } from '../components/ui/NeoButton';
import { ModernLoader } from '../components/ui/ModernLoader';

export function ModernDashboard() {
  const { data: stats, isLoading: statsLoading } = useDashboardStats();
  const { data: shipments, isLoading: shipmentsLoading } = useShipments();

  // Get recent shipments and delivered today
  const recentShipments = shipments?.slice(0, 3) || [];
  const deliveredToday = shipments?.filter(s => {
    const today = new Date().toDateString();
    return s.is_delivered && s.updated_at && new Date(s.updated_at).toDateString() === today;
  }) || [];

  // Modern greeting with better typography
  const getGreeting = () => {
    const hour = new Date().getHours();
    if (hour < 12) return { text: 'Good morning', emoji: '☀️', color: 'from-amber-400 to-orange-500' };
    if (hour < 17) return { text: 'Good afternoon', emoji: '🌤️', color: 'from-blue-400 to-cyan-500' };
    return { text: 'Good evening', emoji: '🌙', color: 'from-purple-400 to-indigo-500' };
  };

  const greeting = getGreeting();

  // Smart insights with better context
  const getSmartInsight = () => {
    if (deliveredToday.length > 0) {
      return { 
        text: `${deliveredToday.length} package${deliveredToday.length > 1 ? 's' : ''} delivered today`,
        icon: '🎉',
        type: 'success' as const
      };
    }
    const inTransit = stats?.in_transit || 0;
    if (inTransit > 0) {
      return { 
        text: `${inTransit} package${inTransit > 1 ? 's are' : ' is'} on the way`,
        icon: '📦',
        type: 'info' as const
      };
    }
    return { 
      text: 'All packages accounted for',
      icon: '✨',
      type: 'neutral' as const
    };
  };

  const insight = getSmartInsight();

  if (statsLoading || shipmentsLoading) {
    return (
      <div className="min-h-[400px] flex items-center justify-center">
        <ModernLoader type="spinner" size="lg" color="blue" />
      </div>
    );
  }

  return (
    <div className="space-y-8 p-6">
      {/* Modern Hero Section */}
      <div className="text-center space-y-4 mb-12">
        <h1 className={`text-6xl font-bold bg-gradient-to-r ${greeting.color} bg-clip-text text-transparent`}>
          {greeting.text} {greeting.emoji}
        </h1>
        
        <p className="text-xl text-slate-600 dark:text-slate-300 flex items-center justify-center gap-2">
          <span className="text-2xl">{insight.icon}</span>
          {insight.text}
        </p>
      </div>

      {/* Modern Bento Grid Layout */}
      <BentoGrid>
        {/* Total Packages - Large card */}
        <BentoCard size="large" gradient="blue">
          <div className="h-full flex flex-col justify-between">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm font-medium text-slate-600 dark:text-slate-300">Total Packages</p>
                <p className="text-4xl font-bold text-slate-900 dark:text-white mt-2">
                  {stats?.total_shipments || 0}
                </p>
              </div>
              <div className="p-3 bg-blue-100 dark:bg-blue-900 rounded-xl">
                <Package className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
            </div>
            
            <div className="mt-6">
              <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
                <TrendingUp className="h-4 w-4" />
                <span>+12% from last month</span>
              </div>
            </div>
          </div>
        </BentoCard>

        {/* In Transit */}
        <BentoCard size="medium" gradient="purple">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-slate-600 dark:text-slate-300">In Transit</p>
              <p className="text-2xl font-bold text-slate-900 dark:text-white mt-1">
                {stats?.in_transit || 0}
              </p>
            </div>
            <div className="p-2 bg-purple-100 dark:bg-purple-900 rounded-lg">
              <Truck className="h-5 w-5 text-purple-600 dark:text-purple-400" />
            </div>
          </div>
        </BentoCard>

        {/* Delivered */}
        <BentoCard size="medium" gradient="green">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-slate-600 dark:text-slate-300">Delivered</p>
              <p className="text-2xl font-bold text-slate-900 dark:text-white mt-1">
                {stats?.delivered || 0}
              </p>
            </div>
            <div className="p-2 bg-green-100 dark:bg-green-900 rounded-lg">
              <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
            </div>
          </div>
        </BentoCard>

        {/* Attention Required */}
        <BentoCard size="medium" gradient="orange">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm font-medium text-slate-600 dark:text-slate-300">Needs Attention</p>
              <p className="text-2xl font-bold text-slate-900 dark:text-white mt-1">
                {stats?.requiring_attention || 0}
              </p>
            </div>
            <div className="p-2 bg-orange-100 dark:bg-orange-900 rounded-lg">
              <AlertTriangle className="h-5 w-5 text-orange-600 dark:text-orange-400" />
            </div>
          </div>
        </BentoCard>
      </BentoGrid>

      {/* Recent Activity with Glass Card */}
      <GlassCard blur="lg" opacity="medium" glow>
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-3">
            <div className="p-2 bg-blue-100 dark:bg-blue-900 rounded-lg">
              <Clock className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            </div>
            <h2 className="text-xl font-semibold text-slate-900 dark:text-white">Recent Activity</h2>
          </div>
          
          <Link to="/shipments">
            <NeoButton variant="secondary" size="sm">
              View All
            </NeoButton>
          </Link>
        </div>

        {recentShipments.length === 0 ? (
          <div className="text-center py-12">
            <div>
              <Package className="mx-auto h-16 w-16 text-slate-400 dark:text-slate-600" />
            </div>
            <h3 className="mt-4 text-lg font-medium text-slate-900 dark:text-white">
              Ready for your first package?
            </h3>
            <p className="mt-2 text-slate-600 dark:text-slate-300 max-w-sm mx-auto">
              Start tracking by adding your first shipment and experience the magic.
            </p>
            <div className="mt-6">
              <Link to="/shipments/new">
                <NeoButton variant="primary" size="md">
                  <Plus className="mr-2 h-4 w-4" />
                  Add Your First Package
                </NeoButton>
              </Link>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {recentShipments.map((shipment) => (
              <div
                key={shipment.id}
                className="flex items-center gap-4 p-4 bg-white/50 dark:bg-slate-800/50 rounded-xl border border-slate-200/50 dark:border-slate-700/50 hover:bg-white/70 dark:hover:bg-slate-800/70 transition-all duration-200"
              >
                <div
                  className={`w-3 h-3 rounded-full ${
                    shipment.is_delivered ? 'bg-green-500' : 'bg-blue-500'
                  }`}
                />
                
                <div className="flex-1">
                  <p className="font-medium text-slate-900 dark:text-white">
                    {shipment.description}
                  </p>
                  <p className="text-sm text-slate-600 dark:text-slate-300">
                    {shipment.tracking_number} • {shipment.carrier.toUpperCase()}
                  </p>
                </div>
                
                <Link to={`/shipments/${shipment.id}`}>
                  <NeoButton variant="secondary" size="sm">
                    Track
                  </NeoButton>
                </Link>
              </div>
            ))}
          </div>
        )}
      </GlassCard>

      {/* Quick Actions */}
      <div className="flex justify-center">
        <Link to="/shipments/new">
          <NeoButton variant="primary" size="lg">
            <Plus className="mr-2 h-5 w-5" />
            <Sparkles className="mr-2 h-5 w-5" />
            Add New Package
          </NeoButton>
        </Link>
      </div>
    </div>
  );
}