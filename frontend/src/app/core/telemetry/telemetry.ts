import { HttpClient } from '@angular/common/http';
import { DestroyRef, inject, Injectable, PLATFORM_ID } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';

export interface TelemetryPayload {
  type: string;
  source: string;
  payload?: Record<string, unknown>;
}

interface TelemetryResponse {
  data: {
    accepted: number;
    status?: string;
  };
}

@Injectable({ providedIn: 'root' })
export class Telemetry {
  private http = inject(HttpClient);
  private platformId = inject(PLATFORM_ID);
  private destroyRef = inject(DestroyRef);

  private buffer: TelemetryPayload[] = [];
  private flushTimer: ReturnType<typeof setInterval> | null = null;
  private disabled = false;

  constructor() {
    if (isPlatformBrowser(this.platformId)) {
      this.flushTimer = setInterval(() => {
        this.flush();
      }, 5000);

      this.destroyRef.onDestroy(() => {
        if (this.flushTimer) {
          clearInterval(this.flushTimer);
        }
        this.flush();
      });
    }
  }

  track(type: string, details?: Record<string, unknown>): void {
    if (!isPlatformBrowser(this.platformId) || this.disabled) {
      return;
    }

    this.buffer.push({
      type,
      source: 'tv-launcher',
      payload: details,
    });

    if (this.buffer.length >= 10) {
      this.flush();
    }
  }

  private flush(): void {
    if (this.buffer.length === 0 || this.disabled) {
      return;
    }

    const batch = [...this.buffer];
    this.buffer = [];

    this.http.post<TelemetryResponse>('/api/v1/telemetry/events', batch).subscribe({
      next: (response) => {
        if (response?.data?.status === 'disabled') {
          this.disabled = true;
          this.buffer = [];
        }
      },
      error: () => {},
    });
  }
}
