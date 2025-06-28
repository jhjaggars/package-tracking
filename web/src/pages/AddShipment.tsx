import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Package } from 'lucide-react';
import { useCreateShipment, useCarriers } from '../hooks/api';
import { Button } from '../components/ui/button';
import type { CreateShipmentRequest } from '../types/api';

export function AddShipment() {
  const navigate = useNavigate();
  const { data: carriers } = useCarriers(true); // Only active carriers
  const createShipmentMutation = useCreateShipment();
  
  const [formData, setFormData] = useState<CreateShipmentRequest>({
    tracking_number: '',
    carrier: '',
    description: '',
  });

  const [errors, setErrors] = useState<Partial<CreateShipmentRequest>>({});

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    // Basic validation
    const newErrors: Partial<CreateShipmentRequest> = {};
    if (!formData.tracking_number.trim()) {
      newErrors.tracking_number = 'Tracking number is required';
    }
    if (!formData.carrier) {
      newErrors.carrier = 'Carrier is required';
    }
    if (!formData.description.trim()) {
      newErrors.description = 'Description is required';
    }

    if (Object.keys(newErrors).length > 0) {
      setErrors(newErrors);
      return;
    }

    try {
      await createShipmentMutation.mutateAsync(formData);
      navigate('/shipments');
    } catch (error) {
      console.error('Failed to create shipment:', error);
    }
  };

  const handleInputChange = (
    field: keyof CreateShipmentRequest,
    value: string
  ) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    // Clear error when user starts typing
    if (errors[field]) {
      setErrors(prev => ({ ...prev, [field]: undefined }));
    }
  };

  return (
    <div className="max-w-2xl mx-auto">
      <div className="space-y-6">
        {/* Header */}
        <div>
          <h2 className="text-2xl font-bold leading-7 text-foreground sm:text-3xl">
            Add New Shipment
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Enter the tracking details for your package
          </p>
        </div>

        {/* Form */}
        <div className="bg-card shadow rounded-lg">
          <div className="px-4 py-5 sm:p-6">
            <form onSubmit={handleSubmit} className="space-y-6">
              {/* Tracking Number */}
              <div>
                <label htmlFor="tracking_number" className="block text-sm font-medium text-foreground">
                  Tracking Number
                </label>
                <div className="mt-1">
                  <input
                    type="text"
                    id="tracking_number"
                    value={formData.tracking_number}
                    onChange={(e) => handleInputChange('tracking_number', e.target.value)}
                    className="block w-full px-3 py-2 border border-input rounded-md shadow-sm placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent sm:text-sm"
                    placeholder="Enter tracking number"
                  />
                  {errors.tracking_number && (
                    <p className="mt-2 text-sm text-red-600">{errors.tracking_number}</p>
                  )}
                </div>
              </div>

              {/* Carrier */}
              <div>
                <label htmlFor="carrier" className="block text-sm font-medium text-foreground">
                  Carrier
                </label>
                <div className="mt-1">
                  <select
                    id="carrier"
                    value={formData.carrier}
                    onChange={(e) => handleInputChange('carrier', e.target.value)}
                    className="block w-full px-3 py-2 border border-input rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent sm:text-sm"
                  >
                    <option value="">Select a carrier</option>
                    {carriers?.map((carrier) => (
                      <option key={carrier.code} value={carrier.code}>
                        {carrier.name}
                      </option>
                    ))}
                  </select>
                  {errors.carrier && (
                    <p className="mt-2 text-sm text-red-600">{errors.carrier}</p>
                  )}
                </div>
              </div>

              {/* Description */}
              <div>
                <label htmlFor="description" className="block text-sm font-medium text-foreground">
                  Description
                </label>
                <div className="mt-1">
                  <input
                    type="text"
                    id="description"
                    value={formData.description}
                    onChange={(e) => handleInputChange('description', e.target.value)}
                    className="block w-full px-3 py-2 border border-input rounded-md shadow-sm placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring focus:border-transparent sm:text-sm"
                    placeholder="Enter package description"
                  />
                  {errors.description && (
                    <p className="mt-2 text-sm text-red-600">{errors.description}</p>
                  )}
                </div>
              </div>

              {/* Submit Button */}
              <div className="flex justify-end space-x-3">
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => navigate('/shipments')}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  disabled={createShipmentMutation.isPending}
                >
                  {createShipmentMutation.isPending ? (
                    'Adding...'
                  ) : (
                    <>
                      <Package className="mr-2 h-4 w-4" />
                      Add Shipment
                    </>
                  )}
                </Button>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}