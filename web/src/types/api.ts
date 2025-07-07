// Base API types matching the Go backend models

export interface Shipment {
  id: number;
  tracking_number: string;
  carrier: string;
  description: string;
  status: string;
  created_at: string;
  updated_at: string;
  expected_delivery?: string;
  is_delivered: boolean;
  last_manual_refresh?: string;
  manual_refresh_count: number;
}

export interface TrackingEvent {
  id: number;
  shipment_id: number;
  timestamp: string;
  location: string;
  status: string;
  description: string;
  created_at: string;
}

export interface Carrier {
  id: number;
  name: string;
  code: string;
  api_endpoint: string;
  active: boolean;
}

export interface RefreshResponse {
  shipment_id: number;
  updated_at: string;
  events_added: number;
  total_events: number;
  events: TrackingEvent[];
}

export interface HealthStatus {
  status: string;
  database: string;
  timestamp: string;
}

// API request types
export interface CreateShipmentRequest {
  tracking_number: string;
  carrier: string;
  description: string;
}

export interface UpdateShipmentRequest {
  description: string;
}

// API error type
export interface APIError {
  code: number;
  message: string;
}

// Dashboard statistics (future API endpoint)
export interface DashboardStats {
  total_shipments: number;
  active_shipments: number;
  in_transit: number;
  delivered: number;
  requiring_attention: number;
}

// Shipment status types
export type ShipmentStatus = 
  | 'pending'
  | 'pre_ship' 
  | 'in_transit'
  | 'out_for_delivery'
  | 'delivered'
  | 'exception'
  | 'returned';

// Carrier codes
export type CarrierCode = 'ups' | 'usps' | 'fedex' | 'dhl';

// Email types
export interface EmailEntry {
  id: number;
  gmail_message_id: string;
  gmail_thread_id: string;
  from: string;
  subject: string;
  date: string;
  body_text: string;
  body_html: string;
  internal_timestamp: string;
  scan_method: string;
  processed_at: string;
  status: string;
  tracking_numbers: string;
  error_message?: string;
  created_at: string;
  updated_at: string;
}

export interface EmailThread {
  id: number;
  gmail_thread_id: string;
  subject: string;
  participants: string;
  message_count: number;
  first_message_date: string;
  last_message_date: string;
  created_at: string;
  updated_at: string;
}

export interface EmailThreadResponse {
  thread: EmailThread;
  emails: EmailEntry[];
}