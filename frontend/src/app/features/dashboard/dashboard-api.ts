import { Injectable } from '@angular/core';
import { httpResource } from '@angular/common/http';

export interface DashboardStatus {
  status: string;
  version: string;
  environment: string;
  maintenance_mode: boolean;
  started_at: string;
  uptime_seconds: number;
}

export interface ResourceCounts {
  total_scenes: number;
  visible_scenes: number;
  total_users: number;
  active_users: number;
  active_connections: number;
}

export interface SystemMetrics {
  goroutines: number;
  memory_alloc_mb: number;
  memory_sys_mb: number;
  cpu_cores: number;
}

export interface DashboardStats {
  status: DashboardStatus;
  resources: ResourceCounts;
  metrics: SystemMetrics;
  updated_at: string;
}

export interface DashboardStatusResponse {
  data: DashboardStatus;
}

export interface DashboardStatsResponse {
  data: DashboardStats;
}

@Injectable({ providedIn: 'root' })
export class DashboardApi {
  status = httpResource<DashboardStatusResponse>(() => '/api/v1/dashboard/status');
  stats = httpResource<DashboardStatsResponse>(() => '/api/v1/dashboard/stats');
}
