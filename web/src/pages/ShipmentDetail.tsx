import { useParams, useNavigate } from 'react-router-dom';
import { RefreshCw, ArrowLeft, Edit, Trash2, Clock } from 'lucide-react';
import { useShipment, useShipmentEvents, useRefreshShipment, useDeleteShipment } from '../hooks/api';
import { Button } from '../components/ui/button';
import { StatusBadge, DateFormatter } from '../components/shared';
import type { TrackingEvent } from '../types/api';
import { sanitizePlainText } from '../lib/sanitize';


function TrackingTimeline({ events }: { events: TrackingEvent[] }) {
  if (events.length === 0) {
    return (
      <div className="text-center py-6">
        <Clock className="mx-auto h-12 w-12 text-muted-foreground" />
        <h3 className="mt-2 text-sm font-medium text-foreground">No tracking events yet</h3>
        <p className="mt-1 text-sm text-muted-foreground">
          Try refreshing to get the latest tracking information.
        </p>
      </div>
    );
  }

  return (
    <div className="flow-root">
      <ul className="-mb-8">
        {events.map((event, eventIdx) => (
          <li key={event.id}>
            <div className="relative pb-8">
              {eventIdx !== events.length - 1 ? (
                <span
                  className="absolute top-4 left-4 -ml-px h-full w-0.5 bg-gray-200"
                  aria-hidden="true"
                />
              ) : null}
              <div className="relative flex space-x-3">
                <div>
                  <span className="h-8 w-8 rounded-full bg-primary flex items-center justify-center ring-8 ring-background">
                    <div className="h-2 w-2 rounded-full bg-white" />
                  </span>
                </div>
                <div className="min-w-0 flex-1 pt-1.5 flex justify-between space-x-4">
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {sanitizePlainText(event.description)}
                    </p>
                    {event.location && (
                      <p className="text-sm text-muted-foreground">
                        {sanitizePlainText(event.location)}
                      </p>
                    )}
                  </div>
                  <div className="text-right text-sm whitespace-nowrap text-muted-foreground">
                    <DateFormatter date={event.timestamp} />
                  </div>
                </div>
              </div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}

export function ShipmentDetail() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const shipmentId = parseInt(id || '0');
  
  const { data: shipment, isLoading: shipmentLoading } = useShipment(shipmentId);
  const { data: events, isLoading: eventsLoading } = useShipmentEvents(shipmentId);
  const refreshMutation = useRefreshShipment();
  const deleteMutation = useDeleteShipment();

  const handleRefresh = async () => {
    try {
      await refreshMutation.mutateAsync(shipmentId);
    } catch (error) {
      console.error('Failed to refresh shipment:', error);
    }
  };

  const handleDelete = async () => {
    if (window.confirm('Are you sure you want to delete this shipment?')) {
      try {
        await deleteMutation.mutateAsync(shipmentId);
        navigate('/shipments');
      } catch (error) {
        console.error('Failed to delete shipment:', error);
      }
    }
  };

  if (shipmentLoading) {
    return (
      <div className="text-center py-8">
        <div className="text-muted-foreground">Loading shipment...</div>
      </div>
    );
  }

  if (!shipment) {
    return (
      <div className="text-center py-8">
        <div className="text-foreground">Shipment not found</div>
        <Button variant="outline" onClick={() => navigate('/shipments')} className="mt-4">
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Shipments
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="md:flex md:items-center md:justify-between">
        <div className="flex-1 min-w-0">
          <div className="flex items-center space-x-3">
            <Button variant="ghost" size="sm" onClick={() => navigate('/shipments')}>
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h2 className="text-2xl font-bold leading-7 text-foreground sm:text-3xl">
                {sanitizePlainText(shipment.description)}
              </h2>
              <p className="mt-1 text-sm text-muted-foreground">
                {shipment.tracking_number} • {shipment.carrier.toUpperCase()}
              </p>
            </div>
          </div>
        </div>
        <div className="mt-4 flex space-x-3 md:mt-0 md:ml-4">
          <Button
            variant="outline"
            onClick={handleRefresh}
            disabled={refreshMutation.isPending}
          >
            <RefreshCw className={`mr-2 h-4 w-4 ${refreshMutation.isPending ? 'animate-spin' : ''}`} />
            {refreshMutation.isPending ? 'Refreshing...' : 'Refresh'}
          </Button>
          <Button variant="outline" size="sm">
            <Edit className="h-4 w-4" />
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Shipment Details */}
      <div className="bg-card shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-card-foreground mb-4">
            Shipment Details
          </h3>
          <dl className="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2">
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Tracking Number</dt>
              <dd className="mt-1 text-sm text-foreground font-mono">{shipment.tracking_number}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Carrier</dt>
              <dd className="mt-1 text-sm text-foreground">{shipment.carrier.toUpperCase()}</dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Status</dt>
              <dd className="mt-1"><StatusBadge status={shipment.status} isDelivered={shipment.is_delivered} /></dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Created</dt>
              <dd className="mt-1 text-sm text-foreground"><DateFormatter date={shipment.created_at} /></dd>
            </div>
            <div>
              <dt className="text-sm font-medium text-muted-foreground">Last Updated</dt>
              <dd className="mt-1 text-sm text-foreground"><DateFormatter date={shipment.updated_at} /></dd>
            </div>
            {shipment.expected_delivery && (
              <div>
                <dt className="text-sm font-medium text-muted-foreground">Expected Delivery</dt>
                <dd className="mt-1 text-sm text-foreground">
                  <DateFormatter date={shipment.expected_delivery} />
                </dd>
              </div>
            )}
            {shipment.last_manual_refresh && (
              <div>
                <dt className="text-sm font-medium text-muted-foreground">Last Manual Refresh</dt>
                <dd className="mt-1 text-sm text-foreground">
                  <DateFormatter date={shipment.last_manual_refresh} />
                  <span className="ml-2 text-xs text-muted-foreground">
                    ({shipment.manual_refresh_count} times)
                  </span>
                </dd>
              </div>
            )}
          </dl>
        </div>
      </div>

      {/* Tracking Timeline */}
      <div className="bg-card shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg leading-6 font-medium text-card-foreground mb-4">
            Tracking Timeline
          </h3>
          {eventsLoading ? (
            <div className="text-center py-4">
              <div className="text-muted-foreground">Loading tracking events...</div>
            </div>
          ) : (
            <TrackingTimeline events={events || []} />
          )}
        </div>
      </div>
    </div>
  );
}