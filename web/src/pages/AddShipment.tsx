import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Package, CheckCircle, AlertTriangle, Truck } from 'lucide-react';
import { useCreateShipment, useCarriers } from '../hooks/api';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
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

  // Smart carrier detection based on tracking number
  const detectCarrier = (trackingNumber: string) => {
    const cleanNumber = trackingNumber.trim().toUpperCase();
    
    // UPS tracking numbers typically start with 1Z
    if (cleanNumber.startsWith('1Z') && cleanNumber.length === 18) {
      return 'ups';
    }
    
    // FedEx patterns
    if (cleanNumber.match(/^\d{12}$/) || cleanNumber.match(/^\d{14}$/) || cleanNumber.match(/^\d{20}$/)) {
      return 'fedex';
    }
    
    // USPS patterns
    if (cleanNumber.match(/^(94|93|92|94|95)\d{20}$/) || cleanNumber.match(/^[A-Z]{2}\d{9}[A-Z]{2}$/)) {
      return 'usps';
    }
    
    // DHL patterns
    if (cleanNumber.match(/^\d{10,11}$/) || cleanNumber.match(/^[A-Z0-9]{10}$/)) {
      return 'dhl';
    }
    
    return null;
  };

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

    // Smart carrier detection
    if (field === 'tracking_number' && value.length > 8) {
      const detectedCarrier = detectCarrier(value);
      if (detectedCarrier && detectedCarrier !== formData.carrier) {
        setFormData(prev => ({ ...prev, carrier: detectedCarrier }));
      }
    }
  };

  return (
    <div className="max-w-2xl mx-auto">
      <div className="space-y-6">
        {/* Header */}
        <div className="space-y-2">
          <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
            <Package className="h-8 w-8" />
            Add New Shipment
          </h1>
          <p className="text-muted-foreground">
            Enter your tracking information to start monitoring your package.
          </p>
        </div>

        {/* Form */}
        <Card>
          <CardHeader>
            <CardTitle>Shipment Details</CardTitle>
            <CardDescription>
              Enter the tracking information for your package.
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-6">
              {/* Tracking Number */}
              <div className="space-y-2">
                <Label htmlFor="tracking_number">
                  Tracking Number
                  {formData.carrier && (
                    <Badge variant="secondary" className="ml-2">
                      Auto-detected {formData.carrier.toUpperCase()}
                    </Badge>
                  )}
                </Label>
                <Input
                  id="tracking_number"
                  value={formData.tracking_number}
                  onChange={(e) => handleInputChange('tracking_number', e.target.value)}
                  placeholder="Enter your tracking number (e.g., 1Z999AA1234567890)"
                />
                {errors.tracking_number && (
                  <p className="text-sm text-destructive flex items-center gap-2">
                    <AlertTriangle className="h-4 w-4" />
                    {errors.tracking_number}
                  </p>
                )}
              </div>

              {/* Carrier */}
              <div className="space-y-3">
                <Label>Carrier</Label>
                <div className="grid grid-cols-2 gap-3">
                  {carriers?.map((carrier) => (
                    <Button
                      key={carrier.code}
                      type="button"
                      variant={formData.carrier === carrier.code ? "default" : "outline"}
                      onClick={() => handleInputChange('carrier', carrier.code)}
                      className="justify-start"
                    >
                      <Truck className="mr-2 h-4 w-4" />
                      {carrier.name}
                    </Button>
                  ))}
                </div>
                {errors.carrier && (
                  <p className="text-sm text-destructive flex items-center gap-2">
                    <AlertTriangle className="h-4 w-4" />
                    {errors.carrier}
                  </p>
                )}
              </div>

              {/* Description */}
              <div className="space-y-2">
                <Label htmlFor="description">Package Description</Label>
                <Input
                  id="description"
                  value={formData.description}
                  onChange={(e) => handleInputChange('description', e.target.value)}
                  placeholder="What's in this package? (e.g., Electronics, Books, Clothing)"
                />
                {errors.description && (
                  <p className="text-sm text-destructive flex items-center gap-2">
                    <AlertTriangle className="h-4 w-4" />
                    {errors.description}
                  </p>
                )}
              </div>

              {/* Action Buttons */}
              <div className="flex justify-end space-x-2">
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
                    <>
                      <div className="mr-2 h-4 w-4 animate-spin rounded-full border-2 border-muted border-r-primary" />
                      Creating...
                    </>
                  ) : (
                    <>
                      <CheckCircle className="mr-2 h-4 w-4" />
                      Add Shipment
                    </>
                  )}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}