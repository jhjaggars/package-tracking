import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Package, Wand2, CheckCircle, AlertTriangle, Sparkles, Truck, Camera } from 'lucide-react';
import { useCreateShipment, useCarriers } from '../hooks/api';
import { Button } from '../components/ui/button';
import confetti from 'canvas-confetti';
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
  const [step, setStep] = useState(1);
  const [showSuccess, setShowSuccess] = useState(false);
  const [smartSuggestions, setSmartSuggestions] = useState<string[]>([]);

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

  // Smart description suggestions
  const generateDescriptionSuggestions = (carrier: string) => {
    const suggestions = {
      ups: ['Electronics Package', 'Clothing Order', 'Books & Media', 'Home & Garden', 'Office Supplies'],
      fedex: ['Express Delivery', 'Business Documents', 'Medical Supplies', 'Fragile Items', 'International Package'],
      usps: ['Online Purchase', 'Gift Package', 'Return Item', 'Documents', 'Small Package'],
      dhl: ['International Order', 'Express Package', 'Business Shipment', 'Important Documents', 'Overseas Purchase']
    };
    return suggestions[carrier as keyof typeof suggestions] || ['Package', 'Delivery', 'Order', 'Shipment'];
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
      triggerSuccess();
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
        setSmartSuggestions(generateDescriptionSuggestions(detectedCarrier));
      }
    }

    // Auto-advance to next step
    if (field === 'tracking_number' && value.length > 10 && step === 1) {
      setTimeout(() => setStep(2), 500);
    }
    if (field === 'carrier' && value && step === 2) {
      setTimeout(() => setStep(3), 300);
    }
  };

  const handleSuggestionClick = (suggestion: string) => {
    setFormData(prev => ({ ...prev, description: suggestion }));
    setSmartSuggestions([]);
  };

  const triggerSuccess = () => {
    setShowSuccess(true);
    confetti({
      particleCount: 100,
      spread: 70,
      origin: { y: 0.6 },
      colors: ['#10B981', '#3B82F6', '#8B5CF6']
    });
    setTimeout(() => {
      navigate('/shipments');
    }, 2000);
  };

  if (showSuccess) {
    return (
      <div className="max-w-2xl mx-auto text-center py-16">
        <div className="mx-auto w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mb-6">
          <CheckCircle className="h-8 w-8 text-green-600" />
        </div>
        <h2 className="text-3xl font-bold text-green-600 mb-4">Package Added Successfully! 🎉</h2>
        <p className="text-muted-foreground mb-8">Your package is now being tracked. We'll keep you updated on its journey!</p>
        <div className="text-sm text-muted-foreground">
          Redirecting to your shipments...
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto">
      <div className="space-y-8">
        {/* Delightful Header */}
        <div>
          <h1 className="text-3xl font-bold leading-7 bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent sm:text-4xl flex items-center gap-3">
            <div>
              <Package className="h-8 w-8 text-blue-600" />
            </div>
            Add New Shipment
          </h1>
          <p className="mt-3 text-lg text-muted-foreground flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-yellow-500" />
            Let's get your package tracked with some magic
          </p>
        </div>

        {/* Smart Progress Indicator */}
        <div className="bg-gradient-to-r from-blue-50 to-purple-50 rounded-xl p-4">
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-blue-600">Progress</span>
            <span className="text-sm text-muted-foreground">{Math.round((step / 3) * 100)}% Complete</span>
          </div>
          <div className="w-full bg-blue-200 rounded-full h-2">
            <div 
              className="bg-gradient-to-r from-blue-600 to-purple-600 h-2 rounded-full"
              style={{ width: `${(step / 3) * 100}%` }}
            />
          </div>
        </div>

        {/* Enhanced Form */}
        <div className="bg-card shadow-xl rounded-2xl border-0 overflow-hidden">
          <div className="px-6 py-8 sm:p-10">
            <form onSubmit={handleSubmit} className="space-y-8">
              {/* Step 1: Tracking Number */}
              <div>
                <div className="flex items-center gap-3 mb-4">
                  <div
                    className={`w-8 h-8 rounded-full flex items-center justify-center ${step >= 1 ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-500'}`}
                  >
                    1
                  </div>
                  <label htmlFor="tracking_number" className="text-lg font-semibold text-foreground">
                    Tracking Number
                  </label>
                  {formData.tracking_number && (
                    <div className="text-green-600">
                      <CheckCircle className="h-5 w-5" />
                    </div>
                  )}
                </div>
                <div className="relative">
                  <input
                    type="text"
                    id="tracking_number"
                    value={formData.tracking_number}
                    onChange={(e) => handleInputChange('tracking_number', e.target.value)}
                    className="block w-full px-4 py-4 border-2 border-input rounded-xl shadow-sm placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-base transition-all duration-200"
                    placeholder="Enter your tracking number (e.g., 1Z999AA1234567890)"
                  />
                  <div 
                    className="absolute right-3 top-1/2 transform -translate-y-1/2"
                    style={{ opacity: formData.tracking_number ? 1 : 0 }}
                  >
                    <Camera className="h-5 w-5 text-muted-foreground cursor-pointer hover:text-blue-600" />
                  </div>
                </div>
                {errors.tracking_number && (
                    <p className="mt-3 text-sm text-red-600 flex items-center gap-2">
                      <AlertTriangle className="h-4 w-4" />
                      {errors.tracking_number}
                    </p>
                  )}
              </div>

              {/* Step 2: Carrier */}
              {step >= 2 && (
                  <div>
                    <div className="flex items-center gap-3 mb-4">
                      <div
                        className={`w-8 h-8 rounded-full flex items-center justify-center ${step >= 2 ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-500'}`}
                      >
                        2
                      </div>
                      <label htmlFor="carrier" className="text-lg font-semibold text-foreground">
                        Carrier
                      </label>
                      {formData.carrier && (
                        <div className="text-green-600">
                          <CheckCircle className="h-5 w-5" />
                        </div>
                      )}
                      {formData.carrier && (
                        <div className="flex items-center gap-2 text-sm text-blue-600 bg-blue-50 px-3 py-1 rounded-full">
                          <Wand2 className="h-4 w-4" />
                          Auto-detected!
                        </div>
                      )}
                    </div>
                    <div className="grid grid-cols-2 gap-4">
                      {carriers?.map((carrier) => (
                        <button
                          key={carrier.code}
                          type="button"
                          onClick={() => handleInputChange('carrier', carrier.code)}
                          className={`p-4 border-2 rounded-xl text-left transition-all duration-200 ${
                            formData.carrier === carrier.code
                              ? 'border-blue-600 bg-blue-50 shadow-md'
                              : 'border-gray-200 hover:border-gray-300 hover:shadow-sm'
                          }`}
                        >
                          <div className="flex items-center gap-3">
                            <div className={`w-6 h-6 rounded-full flex items-center justify-center ${
                              carrier.code === 'ups' ? 'bg-amber-100 text-amber-600' :
                              carrier.code === 'fedex' ? 'bg-purple-100 text-purple-600' :
                              carrier.code === 'usps' ? 'bg-blue-100 text-blue-600' :
                              'bg-gray-100 text-gray-600'
                            }`}>
                              <Truck className="h-4 w-4" />
                            </div>
                            <span className="font-medium">{carrier.name}</span>
                          </div>
                        </button>
                      ))}
                    </div>
                    {errors.carrier && (
                        <p className="mt-3 text-sm text-red-600 flex items-center gap-2">
                          <AlertTriangle className="h-4 w-4" />
                          {errors.carrier}
                        </p>
                      )}
                  </div>
                )}

              {/* Step 3: Description */}
              {step >= 3 && (
                  <div>
                    <div className="flex items-center gap-3 mb-4">
                      <div
                        className={`w-8 h-8 rounded-full flex items-center justify-center ${step >= 3 ? 'bg-blue-600 text-white' : 'bg-gray-200 text-gray-500'}`}
                      >
                        3
                      </div>
                      <label htmlFor="description" className="text-lg font-semibold text-foreground">
                        Package Description
                      </label>
                      {formData.description && (
                        <div className="text-green-600">
                          <CheckCircle className="h-5 w-5" />
                        </div>
                      )}
                    </div>
                    <input
                      type="text"
                      id="description"
                      value={formData.description}
                      onChange={(e) => handleInputChange('description', e.target.value)}
                      className="block w-full px-4 py-4 border-2 border-input rounded-xl shadow-sm placeholder-muted-foreground focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 sm:text-base transition-all duration-200"
                      placeholder="What's in this package? (e.g., Electronics, Books, Clothing)"
                    />
                    
                    {/* Smart Suggestions */}
                      {smartSuggestions.length > 0 && !formData.description && (
                        <div className="mt-3">
                          <p className="text-sm text-muted-foreground mb-2 flex items-center gap-2">
                            <Wand2 className="h-4 w-4" />
                            Smart suggestions:
                          </p>
                          <div className="flex flex-wrap gap-2">
                            {smartSuggestions.map((suggestion) => (
                              <button
                                key={suggestion}
                                type="button"
                                onClick={() => handleSuggestionClick(suggestion)}
                                className="px-3 py-1 bg-blue-50 text-blue-700 rounded-full text-sm hover:bg-blue-100 transition-colors"
                              >
                                {suggestion}
                              </button>
                            ))}
                          </div>
                        </div>
                      )}

                    {errors.description && (
                        <p className="mt-3 text-sm text-red-600 flex items-center gap-2">
                          <AlertTriangle className="h-4 w-4" />
                          {errors.description}
                        </p>
                      )}
                  </div>
                )}

              {/* Action Buttons */}
              <div className="flex justify-end space-x-4 pt-6">
                <div>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={() => navigate('/shipments')}
                    className="px-6 py-3"
                  >
                    Cancel
                  </Button>
                </div>
                <div 
                  className={createShipmentMutation.isPending ? 'pointer-events-none' : ''}
                >
                  <Button
                    type="submit"
                    disabled={createShipmentMutation.isPending}
                    className="bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700 text-white border-0 shadow-lg px-8 py-3"
                  >
                    {createShipmentMutation.isPending ? (
                      <div className="flex items-center gap-2">
                        <div className="w-4 h-4 border-2 border-white border-t-transparent rounded-full animate-spin" />
                        Adding Magic...
                      </div>
                    ) : (
                      <>
                        <Sparkles className="mr-2 h-4 w-4" />
                        Add Shipment
                      </>
                    )}
                  </Button>
                </div>
              </div>
            </form>
          </div>
        </div>
      </div>
    </div>
  );
}