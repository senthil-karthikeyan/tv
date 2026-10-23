import { Component, DestroyRef, inject, signal } from '@angular/core';
import { Router, RouterLink } from '@angular/router';
import { merge } from 'rxjs';
import { throttleTime } from 'rxjs/operators';
import { DashboardApi } from './dashboard-api';
import { LiveUpdates } from '../../core/realtime/live-updates';
import { Telemetry } from '../../core/telemetry/telemetry';
import { FocusGroup, FocusItem } from '../../shared/focus/focus-navigation';

interface ChannelRow {
  name: string;
  quality: string;
  viewers: string;
  status: string;
  chip: 'ok' | 'warn' | 'off';
}

@Component({
  imports: [FocusGroup, FocusItem, RouterLink],
  selector: 'app-dashboard',
  templateUrl: './dashboard.html',
})
export class Dashboard {
  private api = inject(DashboardApi);
  private liveUpdates = inject(LiveUpdates);
  readonly telemetry = inject(Telemetry);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);

  readonly statusResource = this.api.status;
  readonly statsResource = this.api.stats;
  readonly connectionStatus = this.liveUpdates.status;

  readonly viewersNow = signal<number>(12480);
  readonly showLiveToast = signal<boolean>(false);
  readonly liveToastMessage = signal<string>('Viewers updated to 12,731');

  readonly chartBars = [38, 44, 41, 52, 60, 58, 66, 72, 81, 94, 88, 76];

  readonly channels = signal<ChannelRow[]>([
    { name: 'Evening news', quality: 'HD', viewers: '4,210', status: 'Healthy', chip: 'ok' },
    { name: 'Cricket live', quality: 'HD', viewers: '3,880', status: 'Healthy', chip: 'ok' },
    { name: 'Movie of the day', quality: '4K', viewers: '2,150', status: 'Slow start', chip: 'warn' },
    { name: 'Kids zone', quality: 'HD', viewers: '1,420', status: 'Healthy', chip: 'ok' },
    { name: 'Regional music', quality: 'SD', viewers: '820', status: 'Idle', chip: 'off' },
  ]);

  constructor() {
    this.telemetry.track('scene.open', { scene: 'dashboard' });

    const reloadSub = merge(
      this.liveUpdates.on('user'),
      this.liveUpdates.on('scene'),
      this.liveUpdates.on('settings'),
      this.liveUpdates.on('telemetry')
    )
      .pipe(throttleTime(2000, undefined, { leading: true, trailing: true }))
      .subscribe(() => {
        this.api.stats.reload();
        this.viewersNow.set(12731);
        this.showLiveToast.set(true);
        this.telemetry.track('ws.message', { scene: 'viewers.changed' });
      });

    this.destroyRef.onDestroy(() => {
      reloadSub.unsubscribe();
    });
  }

  goBack(): void {
    this.telemetry.track('scene.back', { scene: 'dashboard' });
    this.router.navigateByUrl('/launcher');
  }
}
