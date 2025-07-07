import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Package, Search, Plus, RefreshCw, Eye } from 'lucide-react';
import { useShipments } from '../hooks/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Card, CardContent } from '@/components/ui/card';
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table';
import { ShipmentStatusBadge, formatDateOnly } from '../components/shared';
import { sanitizePlainText } from '../lib/sanitize';


export function ShipmentList() {
  const navigate = useNavigate();
  const { data: shipments, isLoading, refetch, isFetching } = useShipments();
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

  // Handle row click to navigate to shipment detail
  const handleRowClick = (shipmentId: number) => {
    navigate(`/shipments/${shipmentId}`);
  };

  // Handle refresh with feedback
  const handleRefresh = async () => {
    try {
      await refetch();
    } catch (error) {
      console.error('Failed to refresh shipments:', error);
    }
  };

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
          <Button variant="outline" onClick={handleRefresh} disabled={isFetching}>
            <RefreshCw className={`mr-2 h-4 w-4 ${isFetching ? 'animate-spin' : ''}`} />
            {isFetching ? 'Refreshing...' : 'Refresh'}
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
      <Card>
        <CardContent className="p-4">
          <div className="flex flex-col space-y-4 sm:flex-row sm:space-y-0 sm:space-x-4">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder="Search by tracking number or description..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9"
                />
              </div>
            </div>
            
            <div className="flex space-x-2">
              <Select value={filterCarrier || "all"} onValueChange={(value) => setFilterCarrier(value === "all" ? "" : value)}>
                <SelectTrigger className="w-[140px]">
                  <SelectValue placeholder="All Carriers" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Carriers</SelectItem>
                  {carriers.map(carrier => (
                    <SelectItem key={carrier} value={carrier}>
                      {carrier.toUpperCase()}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              
              <Select value={filterStatus || "all"} onValueChange={(value) => setFilterStatus(value === "all" ? "" : value)}>
                <SelectTrigger className="w-[120px]">
                  <SelectValue placeholder="All Status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Status</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="delivered">Delivered</SelectItem>
                  <SelectItem value="in_transit">In Transit</SelectItem>
                  <SelectItem value="exception">Exception</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Shipments Table */}
      <Card>
        <CardContent className="p-0">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
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
              <Table className="table-fixed min-w-[800px]">
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[200px]">Package</TableHead>
                  <TableHead className="w-[180px]">Tracking Number</TableHead>
                  <TableHead className="w-[100px]">Carrier</TableHead>
                  <TableHead className="w-[120px]">Status</TableHead>
                  <TableHead className="w-[100px]">Created</TableHead>
                  <TableHead className="w-[80px] text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredShipments.map((shipment) => (
                  <TableRow 
                    key={shipment.id}
                    className="cursor-pointer hover:bg-muted/50"
                    onClick={() => handleRowClick(shipment.id)}
                  >
                    <TableCell className="font-medium w-[200px] whitespace-normal">
                      <div className="line-clamp-2 max-w-[200px]">
                        {sanitizePlainText(shipment.description)}
                      </div>
                    </TableCell>
                    <TableCell className="w-[180px]">
                      <code className="text-sm text-muted-foreground">
                        {shipment.tracking_number}
                      </code>
                    </TableCell>
                    <TableCell className="w-[100px]">
                      {shipment.carrier.toUpperCase()}
                    </TableCell>
                    <TableCell className="w-[120px]">
                      <ShipmentStatusBadge shipment={shipment} />
                    </TableCell>
                    <TableCell className="text-muted-foreground w-[100px]">
                      {formatDateOnly(shipment.created_at)}
                    </TableCell>
                    <TableCell className="text-right w-[80px]" onClick={(e) => e.stopPropagation()}>
                      <Button variant="ghost" size="sm" asChild>
                        <Link to={`/shipments/${shipment.id}`}>
                          <Eye className="h-4 w-4" />
                        </Link>
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}