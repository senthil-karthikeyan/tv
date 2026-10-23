import { DestroyRef, inject, Injectable, PLATFORM_ID, signal } from '@angular/core';
import { isPlatformBrowser } from '@angular/common';
import { Observable, Subject } from 'rxjs';
import { filter } from 'rxjs/operators';
import { API_CONFIG } from '../http/api-config';

export interface WsEvent<T = unknown> {
  type: string;
  resource: string;
  resource_id?: string;
  operation: string;
  data?: T;
  timestamp: string;
}

export type ConnectionStatus = 'connected' | 'connecting' | 'disconnected';

@Injectable({ providedIn: 'root' })
export class LiveUpdates {
  private config = inject(API_CONFIG);
  private platformId = inject(PLATFORM_ID);
  private destroyRef = inject(DestroyRef);

  private statusSignal = signal<ConnectionStatus>('disconnected');
  readonly status = this.statusSignal.asReadonly();

  private lastEventSignal = signal<WsEvent | null>(null);
  readonly lastEvent = this.lastEventSignal.asReadonly();

  private events$ = new Subject<WsEvent>();

  private socket: WebSocket | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 1000;
  private isDestroyed = false;

  constructor() {
    this.destroyRef.onDestroy(() => {
      this.isDestroyed = true;
      if (this.reconnectTimer) {
        clearTimeout(this.reconnectTimer);
      }
      if (this.socket) {
        this.socket.close();
      }
    });

    if (isPlatformBrowser(this.platformId)) {
      this.connect();
    }
  }

  on(resource: string): Observable<WsEvent> {
    return this.events$.pipe(filter((event) => event.resource === resource));
  }

  private connect(): void {
    if (this.isDestroyed) {
      return;
    }

    this.statusSignal.set('connecting');

    const base = this.config.wsUrl.replace(/^http/, 'ws');
    const url = base.endsWith('/ws') ? base : `${base.replace(/\/$/, '')}/ws`;

    try {
      this.socket = new WebSocket(url);
    } catch {
      this.scheduleReconnect();
      return;
    }

    this.socket.onopen = () => {
      this.statusSignal.set('connected');
      this.reconnectDelay = 1000;
    };

    this.socket.onmessage = (event: MessageEvent<string>) => {
      try {
        const parsed = JSON.parse(event.data) as WsEvent;
        this.lastEventSignal.set(parsed);
        this.events$.next(parsed);
      } catch {}
    };

    this.socket.onclose = () => {
      this.statusSignal.set('disconnected');
      this.scheduleReconnect();
    };

    this.socket.onerror = () => {
      this.statusSignal.set('disconnected');
    };
  }

  private scheduleReconnect(): void {
    if (this.isDestroyed || this.reconnectTimer) {
      return;
    }

    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.connect();
    }, this.reconnectDelay);

    this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
  }
}
