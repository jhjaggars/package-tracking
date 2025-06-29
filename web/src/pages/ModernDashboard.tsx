import { Package, Truck, CheckCircle, AlertTriangle, Plus, Sparkles, TrendingUp, Clock } from 'lucide-react';
import { useDashboardStats, useShipments } from '../hooks/api';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { useState } from 'react';
import confetti from 'canvas-confetti';

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
        <ModernLoader type="neural" size="lg" color="multicolor" />
      </div>
    );
  }

  return (
    <div className="space-y-8 p-6">
      {/* Modern Hero Section */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        className="text-center space-y-4 mb-12"
      >
        <motion.h1 
          className={`text-6xl font-bold bg-gradient-to-r ${greeting.color} bg-clip-text text-transparent`}
          animate={{ scale: [1, 1.02, 1] }}
          transition={{ duration: 2, repeat: Infinity }}
        >
          {greeting.text} {greeting.emoji}
        </motion.h1>
        
        <motion.p 
          className="text-xl text-slate-600 dark:text-slate-300 flex items-center justify-center gap-2"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ delay: 0.5 }}
        >
          <span className="text-2xl">{insight.icon}</span>
          {insight.text}
        </motion.p>
      </motion.div>

      {/* Modern Bento Grid Layout */}
      <BentoGrid>
        {/* Total Packages - Large card */}
        <BentoCard size="large" gradient="blue">
          <div className="h-full flex flex-col justify-between">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-sm font-medium text-slate-600 dark:text-slate-300">Total Packages</p>
                <motion.p 
                  className="text-4xl font-bold text-slate-900 dark:text-white mt-2"
                  animate={{ scale: [1, 1.1, 1] }}
                  transition={{ duration: 0.5 }}
                >
                  {stats?.total_shipments || 0}
                </motion.p>
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
            <motion.div 
              className="p-2 bg-purple-100 dark:bg-purple-900 rounded-lg"
              animate={{ x: [0, 5, 0] }}
              transition={{ duration: 2, repeat: Infinity }}
            >
              <Truck className="h-5 w-5 text-purple-600 dark:text-purple-400" />
            </motion.div>
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
            <motion.div 
              className="p-2 bg-green-100 dark:bg-green-900 rounded-lg"
              animate={{ scale: [1, 1.2, 1] }}
              transition={{ duration: 1.5, repeat: Infinity }}
            >
              <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
            </motion.div>
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
            <motion.div 
              className="p-2 bg-orange-100 dark:bg-orange-900 rounded-lg"
              animate={{ rotate: [0, 10, -10, 0] }}
              transition={{ duration: 2, repeat: Infinity }}
            >
              <AlertTriangle className="h-5 w-5 text-orange-600 dark:text-orange-400" />
            </motion.div>
          </div>
        </BentoCard>
      </BentoGrid>

      {/* Recent Activity with Glass Card */}
      <GlassCard blur="lg" opacity="medium" glow hover3d>
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
          <motion.div 
            className="text-center py-12"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
          >
            <motion.div
              animate={{ y: [0, -10, 0] }}
              transition={{ duration: 2, repeat: Infinity }}
            >
              <Package className="mx-auto h-16 w-16 text-slate-400 dark:text-slate-600" />
            </motion.div>
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
          </motion.div>
        ) : (
          <div className="space-y-4">
            {recentShipments.map((shipment, index) => (
              <motion.div
                key={shipment.id}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.1 }}
                className="flex items-center gap-4 p-4 bg-white/50 dark:bg-slate-800/50 rounded-xl border border-slate-200/50 dark:border-slate-700/50 hover:bg-white/70 dark:hover:bg-slate-800/70 transition-all duration-200"
              >
                <motion.div
                  className={`w-3 h-3 rounded-full ${
                    shipment.is_delivered ? 'bg-green-500' : 'bg-blue-500'
                  }`}
                  animate={!shipment.is_delivered ? {
                    scale: [1, 1.2, 1],
                    opacity: [1, 0.7, 1]
                  } : {}}
                  transition={{ duration: 2, repeat: Infinity }}
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
              </motion.div>
            ))}
          </div>
        )}
      </GlassCard>

      {/* Quick Actions */}
      <motion.div 
        className="flex justify-center"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ delay: 0.8 }}
      >
        <Link to="/shipments/new">
          <NeoButton variant="primary" size="lg">
            <Plus className="mr-2 h-5 w-5" />
            <Sparkles className="mr-2 h-5 w-5" />
            Add New Package
          </NeoButton>
        </Link>
      </motion.div>
    </div>
  );
}