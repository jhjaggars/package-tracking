import axios from 'axios';
import type { AxiosResponse, AxiosError } from 'axios';
import { logger } from '../lib/logger';
import type {
  Shipment,
  TrackingEvent,
  Carrier,
  CreateShipmentRequest,
  UpdateShipmentRequest,
  RefreshResponse,
  HealthStatus,
  APIError,
  DashboardStats,
  EmailEntry,
  EmailThreadResponse
} from '../types/api';

// Create axios instance with base configuration
const api = axios.create({
  baseURL: import.meta.env.DEV ? 'http://localhost:8080/api' : '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for logging (development only)
api.interceptors.request.use(
  (config) => {
    logger.debug(`API Request: ${config.method?.toUpperCase()} ${config.url}`);
    return config;
  },
  (error) => {
    logger.error('API Request Error:', error);
    return Promise.reject(error);
  }
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response: AxiosResponse) => {
    logger.debug(`API Response: ${response.status} ${response.config.url}`);
    return response;
  },
  (error: AxiosError<APIError>) => {
    logger.error('API Response Error:', error);
    
    // Transform axios error to our APIError format
    if (error.response?.data) {
      // Server returned an error response
      throw error.response.data;
    } else if (error.request) {
      // Network error
      throw {
        code: 0,
        message: 'Network error - please check your connection'
      } as APIError;
    } else {
      // Request setup error
      throw {
        code: 0,
        message: error.message || 'Unknown error occurred'
      } as APIError;
    }
  }
);

// API service functions
export const apiService = {
  // Health check
  async getHealth(): Promise<HealthStatus> {
    const response = await api.get<HealthStatus>('/health');
    return response.data;
  },

  // Shipments
  async getShipments(): Promise<Shipment[]> {
    const response = await api.get<Shipment[]>('/shipments');
    return response.data;
  },

  async getShipment(id: number): Promise<Shipment> {
    const response = await api.get<Shipment>(`/shipments/${id}`);
    return response.data;
  },

  async createShipment(data: CreateShipmentRequest): Promise<Shipment> {
    const response = await api.post<Shipment>('/shipments', data);
    return response.data;
  },

  async updateShipment(id: number, data: UpdateShipmentRequest): Promise<Shipment> {
    const response = await api.put<Shipment>(`/shipments/${id}`, data);
    return response.data;
  },

  async deleteShipment(id: number): Promise<void> {
    await api.delete(`/shipments/${id}`);
  },

  // Tracking events
  async getShipmentEvents(shipmentId: number): Promise<TrackingEvent[]> {
    const response = await api.get<TrackingEvent[]>(`/shipments/${shipmentId}/events`);
    return response.data;
  },

  // Manual refresh
  async refreshShipment(shipmentId: number): Promise<RefreshResponse> {
    const response = await api.post<RefreshResponse>(`/shipments/${shipmentId}/refresh`);
    return response.data;
  },

  // Carriers
  async getCarriers(activeOnly = false): Promise<Carrier[]> {
    const response = await api.get<Carrier[]>('/carriers', {
      params: activeOnly ? { active: 'true' } : {}
    });
    return response.data;
  },

  // Dashboard stats
  async getDashboardStats(): Promise<DashboardStats> {
    const response = await api.get<DashboardStats>('/dashboard/stats');
    return response.data;
  },

  // Emails
  async getShipmentEmails(shipmentId: number): Promise<EmailEntry[]> {
    const response = await api.get<EmailEntry[]>(`/shipments/${shipmentId}/emails`);
    return response.data;
  },

  async getEmailThread(threadId: string): Promise<EmailThreadResponse> {
    const response = await api.get<EmailThreadResponse>(`/emails/${threadId}/thread`);
    return response.data;
  },

  async getEmailBody(emailId: string): Promise<{ plain_text: string; html_text: string; subject: string; from: string; date: string }> {
    const response = await api.get<{ plain_text: string; html_text: string; subject: string; from: string; date: string }>(`/emails/${emailId}/body`);
    return response.data;
  },
};

export default api;