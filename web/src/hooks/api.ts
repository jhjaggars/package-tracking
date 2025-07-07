import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiService } from '../services/api';
import type {
  CreateShipmentRequest,
  UpdateShipmentRequest,
  RefreshResponse,
  APIError
} from '../types/api';

// Query keys for cache management
export const queryKeys = {
  health: ['health'],
  shipments: ['shipments'],
  shipment: (id: number) => ['shipments', id],
  shipmentEvents: (id: number) => ['shipments', id, 'events'],
  carriers: ['carriers'],
  dashboardStats: ['dashboard', 'stats'],
} as const;

// Health check hook
export function useHealth() {
  return useQuery({
    queryKey: queryKeys.health,
    queryFn: apiService.getHealth,
    refetchInterval: 5 * 60 * 1000, // Refetch every 5 minutes
    retry: 3,
  });
}

// Shipments hooks
export function useShipments() {
  return useQuery({
    queryKey: queryKeys.shipments,
    queryFn: apiService.getShipments,
    refetchInterval: 2 * 60 * 1000, // Refetch every 2 minutes
  });
}

export function useShipment(id: number) {
  return useQuery({
    queryKey: queryKeys.shipment(id),
    queryFn: () => apiService.getShipment(id),
    enabled: !!id,
  });
}

export function useShipmentEvents(shipmentId: number) {
  return useQuery({
    queryKey: queryKeys.shipmentEvents(shipmentId),
    queryFn: () => apiService.getShipmentEvents(shipmentId),
    enabled: !!shipmentId,
  });
}

// Carriers hook
export function useCarriers(activeOnly = false) {
  return useQuery({
    queryKey: [...queryKeys.carriers, activeOnly],
    queryFn: () => apiService.getCarriers(activeOnly),
    staleTime: 10 * 60 * 1000, // Carriers don't change often, cache for 10 minutes
  });
}

// Dashboard stats hook
export function useDashboardStats() {
  return useQuery({
    queryKey: queryKeys.dashboardStats,
    queryFn: apiService.getDashboardStats,
    refetchInterval: 5 * 60 * 1000, // Refetch every 5 minutes
  });
}

// Mutation hooks
export function useCreateShipment() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (data: CreateShipmentRequest) => apiService.createShipment(data),
    onSuccess: () => {
      // Invalidate and refetch shipments list
      queryClient.invalidateQueries({ queryKey: queryKeys.shipments });
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboardStats });
    },
    onError: (error: APIError) => {
      console.error('Failed to create shipment:', error);
    },
  });
}

export function useUpdateShipment() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: UpdateShipmentRequest }) =>
      apiService.updateShipment(id, data),
    onSuccess: (updatedShipment) => {
      // Update the specific shipment in cache
      queryClient.setQueryData(
        queryKeys.shipment(updatedShipment.id),
        updatedShipment
      );
      // Invalidate shipments list to reflect changes
      queryClient.invalidateQueries({ queryKey: queryKeys.shipments });
    },
    onError: (error: APIError) => {
      console.error('Failed to update shipment:', error);
    },
  });
}

export function useDeleteShipment() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (id: number) => apiService.deleteShipment(id),
    onSuccess: (_, deletedId) => {
      // Remove from cache
      queryClient.removeQueries({ queryKey: queryKeys.shipment(deletedId) });
      queryClient.removeQueries({ queryKey: queryKeys.shipmentEvents(deletedId) });
      // Invalidate shipments list
      queryClient.invalidateQueries({ queryKey: queryKeys.shipments });
      queryClient.invalidateQueries({ queryKey: queryKeys.dashboardStats });
    },
    onError: (error: APIError) => {
      console.error('Failed to delete shipment:', error);
    },
  });
}

export function useRefreshShipment() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: (shipmentId: number) => apiService.refreshShipment(shipmentId),
    onSuccess: (data: RefreshResponse) => {
      // Update events cache
      queryClient.setQueryData(
        queryKeys.shipmentEvents(data.shipment_id),
        data.events
      );
      // Invalidate shipment to get updated status
      queryClient.invalidateQueries({ 
        queryKey: queryKeys.shipment(data.shipment_id) 
      });
      // Invalidate shipments list to reflect status changes
      queryClient.invalidateQueries({ queryKey: queryKeys.shipments });
    },
    onError: (error: APIError) => {
      console.error('Failed to refresh shipment:', error);
    },
  });
}

// Email hooks
export function useShipmentEmails(shipmentId: number) {
  return useQuery({
    queryKey: ['shipments', shipmentId, 'emails'],
    queryFn: () => apiService.getShipmentEmails(shipmentId),
    enabled: !!shipmentId,
  });
}

export function useEmailThread(threadId: string) {
  return useQuery({
    queryKey: ['emails', threadId, 'thread'],
    queryFn: () => apiService.getEmailThread(threadId),
    enabled: !!threadId,
  });
}

export function useEmailBody(emailId: string) {
  return useQuery({
    queryKey: ['emails', emailId, 'body'],
    queryFn: () => apiService.getEmailBody(emailId),
    enabled: !!emailId,
  });
}