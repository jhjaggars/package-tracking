import type { Shipment } from '../../types/api';

interface StatusBadgeProps {
  status: string;
  isDelivered: boolean;
}

export function StatusBadge({ status, isDelivered }: StatusBadgeProps) {
  if (isDelivered) {
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
  } as const;
  
  const colorClass = statusColors[status as keyof typeof statusColors] || 'bg-gray-100 text-gray-800';
  
  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${colorClass}`}>
      {status.replace('_', ' ').replace(/\b\w/g, l => l.toUpperCase())}
    </span>
  );
}

// Convenience component for shipment objects
export function ShipmentStatusBadge({ shipment }: { shipment: Shipment }) {
  return <StatusBadge status={shipment.status} isDelivered={shipment.is_delivered} />;
}