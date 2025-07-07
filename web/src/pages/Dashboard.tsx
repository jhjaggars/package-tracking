import { Package, Truck, CheckCircle, AlertTriangle, Plus, Clock, MapPin } from 'lucide-react';
import { useDashboardStats, useShipments } from '../hooks/api';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Skeleton } from '@/components/ui/skeleton';
import { Link } from 'react-router-dom';
import { sanitizePlainText } from '../lib/sanitize';

function StatCard({ 
  title, 
  value, 
  icon: Icon, 
  description,
  loading = false,
}: {
  title: string;
  value: number | string;
  icon: React.ElementType;
  description?: string;
  loading?: boolean;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">
          {title}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">
          {loading ? (
            <Skeleton className="h-8 w-16" data-testid="stat-skeleton" />
          ) : (
            value
          )}
        </div>
        {description && (
          <p className="text-xs text-muted-foreground">
            {description}
          </p>
        )}
      </CardContent>
    </Card>
  );
}

export function Dashboard() {
  const { data: stats, isLoading: statsLoading } = useDashboardStats();
  const { data: shipments, isLoading: shipmentsLoading } = useShipments();

  // Get recent shipments (last 5)
  const recentShipments = shipments?.slice(0, 5) || [];
  const deliveredToday = shipments?.filter(s => {
    const today = new Date().toDateString();
    return s.is_delivered && s.updated_at && new Date(s.updated_at).toDateString() === today;
  }) || [];

  // Smart insights
  const getSmartInsight = () => {
    if (deliveredToday.length > 0) {
      return `${deliveredToday.length} package${deliveredToday.length > 1 ? 's' : ''} delivered today`;
    }
    const inTransit = stats?.in_transit || 0;
    if (inTransit > 0) {
      return `${inTransit} package${inTransit > 1 ? 's are' : ' is'} on the way`;
    }
    return 'All packages accounted for';
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="space-y-1">
          <h1 className="text-3xl font-bold tracking-tight">
            Dashboard
          </h1>
          <p className="text-muted-foreground">
            {getSmartInsight()}
          </p>
        </div>
        <Button asChild>
          <Link to="/shipments/new">
            <Plus className="mr-2 h-4 w-4" />
            Add Shipment
          </Link>
        </Button>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        <StatCard
          title="Total Shipments"
          value={stats?.total_shipments || 0}
          icon={Package}
          loading={statsLoading}
        />
        <StatCard
          title="In Transit"
          value={stats?.in_transit || 0}
          icon={Truck}
          description="Currently shipping"
          loading={statsLoading}
        />
        <StatCard
          title="Delivered"
          value={stats?.delivered || 0}
          icon={CheckCircle}
          description="Successfully delivered"
          loading={statsLoading}
        />
        <StatCard
          title="Requiring Attention"
          value={stats?.requiring_attention || 0}
          icon={AlertTriangle}
          description="Issues or exceptions"
          loading={statsLoading}
        />
      </div>

      {/* Recent Shipments */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div className="flex items-center space-x-2">
              <Clock className="h-5 w-5" />
              <CardTitle>Recent Activity</CardTitle>
            </div>
            <Button variant="outline" size="sm" asChild>
              <Link to="/shipments">View All</Link>
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {shipmentsLoading ? (
            <div className="flex items-center justify-center py-8">
              <div className="flex items-center space-x-2">
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-muted border-r-primary" />
                <span className="text-muted-foreground">Loading shipments...</span>
              </div>
            </div>
          ) : recentShipments.length === 0 ? (
            <div className="text-center py-12">
              <Package className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-4 text-lg font-medium">
                No shipments yet
              </h3>
              <p className="mt-2 text-muted-foreground">
                Get started by adding your first tracking number.
              </p>
              <div className="mt-6">
                <Button asChild>
                  <Link to="/shipments/new">
                    <Plus className="mr-2 h-4 w-4" />
                    Add Your First Shipment
                  </Link>
                </Button>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              {recentShipments.map((shipment) => (
                <div
                  key={shipment.id}
                  className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50"
                >
                  <div className="flex items-center space-x-3">
                    <div 
                      className={`w-2 h-2 rounded-full ${
                        shipment.is_delivered 
                          ? 'bg-green-500' 
                          : shipment.status === 'exception' 
                          ? 'bg-red-500'
                          : 'bg-blue-500'
                      }`}
                    />
                    <div className="min-w-0 flex-1">
                      <p className="text-sm font-medium truncate">
                        {sanitizePlainText(shipment.description)}
                      </p>
                      <div className="flex items-center gap-2 mt-1">
                        <p className="text-sm text-muted-foreground">
                          {shipment.tracking_number}
                        </p>
                        <Badge variant="outline" className="text-xs">
                          {shipment.carrier.toUpperCase()}
                        </Badge>
                        {shipment.is_delivered && (
                          <Badge variant="default" className="text-xs">
                            <CheckCircle className="mr-1 h-3 w-3" />
                            Delivered
                          </Badge>
                        )}
                      </div>
                    </div>
                  </div>
                  <Button variant="ghost" size="sm" asChild>
                    <Link to={`/shipments/${shipment.id}`}>
                      <MapPin className="mr-1 h-3 w-3" />
                      Track
                    </Link>
                  </Button>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}