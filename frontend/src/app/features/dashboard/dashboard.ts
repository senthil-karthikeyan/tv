import { Component, DestroyRef, inject } from '@angular/core';
import { Router } from '@angular/router';
import { merge } from 'rxjs';
import { throttleTime } from 'rxjs/operators';
import { DashboardApi } from './dashboard-api';
import { LiveUpdates } from '../../core/realtime/live-updates';
import { Telemetry } from '../../core/telemetry/telemetry';
import { FocusGroup, FocusItem } from '../../shared/focus/focus-navigation';

@Component({
  imports: [FocusGroup, FocusItem],
  selector: 'app-dashboard',
  templateUrl: './dashboard.html',
})
export class Dashboard {
  private api = inject(DashboardApi);
  private liveUpdates = inject(LiveUpdates);
  private telemetry = inject(Telemetry);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);

  readonly statusResource = this.api.status;
  readonly statsResource = this.api.stats;
  readonly connectionStatus = this.liveUpdates.status;

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
      });

    this.destroyRef.onDestroy(() => {
      reloadSub.unsubscribe();
    });
  }

  goBack(): void {
    this.router.navigateByUrl('/launcher');
  }
}
