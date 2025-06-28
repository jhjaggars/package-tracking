import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Package, Search, Plus, RefreshCw, Eye } from 'lucide-react';
import { useShipments } from '../hooks/api';
import { Button } from '../components/ui/button';
import type { Shipment } from '../types/api';
import { sanitizePlainText } from '../lib/sanitize';

function getStatusBadge(shipment: Shipment) {
  if (shipment.is_delivered) {
    return (
      <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
        Delivered
      </span>
    );
  }
  
  const statusColors = {
    pending: 'bg-gray-100 text-gray-800',
    pre_ship: 'bg-blue-100 text-blue-800',
    in_transit: 'bg-blue-100 text-blue-800',
    out_for_delivery: 'bg-yellow-100 text-yellow-800',
    exception: 'bg-red-100 text-red-800',
    returned: 'bg-red-100 text-red-800',
  };
  
  const colorClass = statusColors[shipment.status as keyof typeof statusColors] || 'bg-gray-100 text-gray-800';
  
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colorClass}`}>
      {shipment.status.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())}
    </span>
  );
}

export function ShipmentList() {
  const { data: shipments, isLoading, refetch } = useShipments();
  const [searchTerm, setSearchTerm] = useState('');
  const [filterCarrier, setFilterCarrier] = useState('');
  const [filterStatus, setFilterStatus] = useState('');

  // Filter shipments based on search and filters
  const filteredShipments = shipments?.filter(shipment => {
    const matchesSearch = !searchTerm || 
      shipment.tracking_number.toLowerCase().includes(searchTerm.toLowerCase()) ||
      shipment.description.toLowerCase().includes(searchTerm.toLowerCase());
    
    const matchesCarrier = !filterCarrier || shipment.carrier === filterCarrier;
    
    const matchesStatus = !filterStatus || 
      (filterStatus === 'delivered' && shipment.is_delivered) ||
      (filterStatus === 'active' && !shipment.is_delivered) ||
      (filterStatus !== 'delivered' && filterStatus !== 'active' && shipment.status === filterStatus);
    
    return matchesSearch && matchesCarrier && matchesStatus;
  }) || [];

  // Get unique carriers for filter
  const carriers = [...new Set(shipments?.map(s => s.carrier) || [])];

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="md:flex md:items-center md:justify-between">
        <div className="flex-1 min-w-0">
          <h2 className="text-2xl font-bold leading-7 text-foreground sm:text-3xl sm:truncate">
            Shipments
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Manage and track all your packages
          </p>
        </div>
        <div className="mt-4 flex space-x-3 md:mt-0 md:ml-4">
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Refresh
          </Button>
          <Button asChild>
            <Link to="/shipments/new">
              <Plus className="mr-2 h-4 w-4" />
              Add Shipment
            </Link>
          </Button>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-card p-4 rounded-lg border space-y-4 sm:space-y-0 sm:flex sm:items-center sm:space-x-4">
        <div className="flex-1 min-w-0">
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Search className="h-5 w-5 text-muted-foreground" />
            </div>
            <input
              type="text"
              placeholder="Search by tracking number or description..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="block w-full pl-10 pr-3 py-2 border border-input rounded-md placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent sm:text-sm"
            />
          </div>
        </div>
        
        <div className="flex space-x-3">
          <select
            value={filterCarrier}
            onChange={(e) => setFilterCarrier(e.target.value)}
            className="block w-full px-3 py-2 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent sm:text-sm"
          >
            <option value="">All Carriers</option>
            {carriers.map(carrier => (
              <option key={carrier} value={carrier}>
                {carrier.toUpperCase()}
              </option>
            ))}
          </select>
          
          <select
            value={filterStatus}
            onChange={(e) => setFilterStatus(e.target.value)}
            className="block w-full px-3 py-2 border border-input rounded-md focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent sm:text-sm"
          >
            <option value="">All Status</option>
            <option value="active">Active</option>
            <option value="delivered">Delivered</option>
            <option value="in_transit">In Transit</option>
            <option value="exception">Exception</option>
          </select>
        </div>
      </div>

      {/* Shipments Table */}
      <div className="bg-card shadow rounded-lg overflow-hidden">
        {isLoading ? (
          <div className="text-center py-8">
            <div className="text-muted-foreground">Loading shipments...</div>
          </div>
        ) : filteredShipments.length === 0 ? (
          <div className="text-center py-8">
            <Package className="mx-auto h-12 w-12 text-muted-foreground" />
            <h3 className="mt-2 text-sm font-medium text-foreground">
              {shipments?.length === 0 ? 'No shipments yet' : 'No matching shipments'}
            </h3>
            <p className="mt-1 text-sm text-muted-foreground">
              {shipments?.length === 0 
                ? 'Get started by adding your first shipment.'
                : 'Try adjusting your search or filters.'
              }
            </p>
            {shipments?.length === 0 && (
              <div className="mt-6">
                <Button asChild>
                  <Link to="/shipments/new">
                    <Plus className="mr-2 h-4 w-4" />
                    Add Shipment
                  </Link>
                </Button>
              </div>
            )}
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-border">
              <thead className="bg-muted/50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Package
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Tracking Number
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Carrier
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Created
                  </th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-muted-foreground uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border">
                {filteredShipments.map((shipment) => (
                  <tr key={shipment.id} className="hover:bg-muted/30">
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm font-medium text-foreground">
                        {sanitizePlainText(shipment.description)}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-muted-foreground font-mono">
                        {shipment.tracking_number}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <div className="text-sm text-foreground">
                        {shipment.carrier.toUpperCase()}
                      </div>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      {getStatusBadge(shipment)}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-muted-foreground">
                      {new Date(shipment.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                      <Button variant="ghost" size="sm" asChild>
                        <Link to={`/shipments/${shipment.id}`}>
                          <Eye className="h-4 w-4" />
                        </Link>
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}