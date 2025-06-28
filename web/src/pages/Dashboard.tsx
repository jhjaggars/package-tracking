import { Package, Truck, CheckCircle, AlertTriangle, Plus } from 'lucide-react';
import { useDashboardStats, useShipments } from '../hooks/api';
import { Button } from '../components/ui/button';
import { Link } from 'react-router-dom';

function StatCard({ 
  title, 
  value, 
  icon: Icon, 
  description,
  loading = false 
}: {
  title: string;
  value: number | string;
  icon: React.ElementType;
  description?: string;
  loading?: boolean;
}) {
  return (
    <div className="bg-card p-6 rounded-lg border shadow-sm">
      <div className="flex items-center">
        <div className="flex-shrink-0">
          <Icon className="h-8 w-8 text-primary" />
        </div>
        <div className="ml-5 w-0 flex-1">
          <dl>
            <dt className="text-sm font-medium text-muted-foreground truncate">
              {title}
            </dt>
            <dd className="text-3xl font-semibold text-card-foreground">
              {loading ? '...' : value}
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

  // Get recent shipments (last 5)
  const recentShipments = shipments?.slice(0, 5) || [];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="md:flex md:items-center md:justify-between">
        <div className="flex-1 min-w-0">
          <h2 className="text-2xl font-bold leading-7 text-foreground sm:text-3xl sm:truncate">
            Dashboard
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Overview of your package tracking activity
          </p>
        </div>
        <div className="mt-4 flex md:mt-0 md:ml-4">
          <Button asChild>
            <Link to="/shipments/new">
              <Package className="mr-2 h-4 w-4" />
              Add Shipment
            </Link>
          </Button>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-4">
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
      <div className="bg-card shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-lg leading-6 font-medium text-card-foreground">
              Recent Shipments
            </h3>
            <Button variant="outline" size="sm" asChild>
              <Link to="/shipments">View All</Link>
            </Button>
          </div>
          
          {shipmentsLoading ? (
            <div className="text-center py-4">
              <div className="text-muted-foreground">Loading shipments...</div>
            </div>
          ) : recentShipments.length === 0 ? (
            <div className="text-center py-8">
              <Package className="mx-auto h-12 w-12 text-muted-foreground" />
              <h3 className="mt-2 text-sm font-medium text-foreground">
                No shipments yet
              </h3>
              <p className="mt-1 text-sm text-muted-foreground">
                Get started by adding your first shipment.
              </p>
              <div className="mt-6">
                <Button asChild>
                  <Link to="/shipments/new">
                    <Plus className="mr-2 h-4 w-4" />
                    Add Shipment
                  </Link>
                </Button>
              </div>
            </div>
          ) : (
            <div className="space-y-3">
              {recentShipments.map((shipment) => (
                <div
                  key={shipment.id}
                  className="flex items-center justify-between p-3 border rounded-md hover:bg-accent/50 transition-colors"
                >
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center space-x-3">
                      <div className="flex-shrink-0">
                        <div className={`w-2 h-2 rounded-full ${
                          shipment.is_delivered 
                            ? 'bg-green-500' 
                            : shipment.status === 'exception' 
                            ? 'bg-red-500'
                            : 'bg-blue-500'
                        }`} />
                      </div>
                      <div className="flex-1 min-w-0">
                        <p className="text-sm font-medium text-foreground truncate">
                          {shipment.description}
                        </p>
                        <p className="text-sm text-muted-foreground">
                          {shipment.tracking_number} • {shipment.carrier.toUpperCase()}
                        </p>
                      </div>
                    </div>
                  </div>
                  <div className="flex-shrink-0">
                    <Button variant="ghost" size="sm" asChild>
                      <Link to={`/shipments/${shipment.id}`}>View</Link>
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