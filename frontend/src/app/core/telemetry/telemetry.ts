import { HttpClient } from '@angular/common/http';
import { DestroyRef, inject, Injectable, PLATFORM_ID, signal } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';

export interface TelemetryPayload {
  type: string;
  source: string;
  payload?: Record<string, unknown>;
}

export interface TelemetryEventItem {
  id: number;
  time: string;
  text: string;
  status: number;
  isNew?: boolean;
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

  readonly recentEvents = signal<TelemetryEventItem[]>([
    { id: 1, time: '11:33:09', text: 'scene.open · dashboard', status: 202, isNew: true },
    { id: 2, time: '11:33:02', text: 'launcher.focus · dashboard', status: 202 },
    { id: 3, time: '11:32:58', text: 'launcher.focus · users', status: 202 },
    { id: 4, time: '11:32:41', text: 'scene.back · home', status: 202 },
    { id: 5, time: '11:32:30', text: 'scene.open · home', status: 202 },
  ]);

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
    const now = new Date();
    const timeStr = [
      now.getHours().toString().padStart(2, '0'),
      now.getMinutes().toString().padStart(2, '0'),
      now.getSeconds().toString().padStart(2, '0'),
    ].join(':');

    const detailText = details?.['scene']
      ? String(details['scene'])
      : details?.['user']
      ? String(details['user'])
      : '';
    const textStr = detailText ? `${type} · ${detailText}` : type;

    this.recentEvents.update((events) => [
      { id: Date.now(), time: timeStr, text: textStr, status: 202, isNew: true },
      ...events.map((e) => ({ ...e, isNew: false })).slice(0, 5),
    ]);

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
