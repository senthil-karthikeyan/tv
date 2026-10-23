import { Component, computed, inject } from '@angular/core';
import { toSignal } from '@angular/core/rxjs-interop';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { Telemetry } from '../../core/telemetry/telemetry';
import { FocusGroup, FocusItem } from '../../shared/focus/focus-navigation';

@Component({
  imports: [FocusGroup, FocusItem, RouterLink],
  selector: 'app-dynamic-page',
  templateUrl: './dynamic-page.html',
})
export class DynamicPage {
  private router = inject(Router);
  private route = inject(ActivatedRoute);
  private telemetry = inject(Telemetry);

  private urlSegments = toSignal(this.route.url, {
    initialValue: this.route.snapshot.url,
  });

  readonly path = computed(() => {
    const segments = this.urlSegments();
    if (segments && segments.length > 0) {
      return segments.map((s) => s.path).join('/');
    }
    const cleanUrl = this.router.url.split('?')[0].split('#')[0].replace(/^\/+|\/+$/g, '');
    return cleanUrl || 'page';
  });

  readonly pageName = computed(() => {
    const p = this.path();
    const last = p.split('/').pop() || 'Page';
    return last
      .split(/[-_]/)
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ');
  });

  constructor() {
    this.telemetry.track('scene.open', { scene: this.path() });
  }

  goBack(): void {
    this.router.navigateByUrl('/launcher');
  }
}
