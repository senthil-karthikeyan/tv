import { Component, computed, DestroyRef, effect, inject, signal } from '@angular/core';
import { RouterLink } from '@angular/router';
import { Scene, SceneApi } from './scene-api';
import { LiveUpdates, WsEvent } from '../../core/realtime/live-updates';
import { Telemetry } from '../../core/telemetry/telemetry';
import { FocusGroup, FocusItem } from '../../shared/focus/focus-navigation';

const ACCENTS = ['#F5D547', '#3FD0E0', '#E266C8', '#4CD38A', '#6F8CFF'];

@Component({
  imports: [FocusGroup, FocusItem, RouterLink],
  selector: 'app-launcher',
  templateUrl: './launcher.html',
})
export class Launcher {
  private api = inject(SceneApi);
  private liveUpdates = inject(LiveUpdates);
  readonly telemetry = inject(Telemetry);
  private destroyRef = inject(DestroyRef);

  readonly scenesResource = this.api.scenes;
  private rawScenes = signal<Scene[]>([]);
  readonly connectionStatus = this.liveUpdates.status;
  readonly lastUpdated = signal<string>('11:32:30');

  readonly scenes = computed(() =>
    this.rawScenes()
      .filter((s) => s.visible)
      .sort((a, b) => a.order - b.order)
  );

  constructor() {
    effect(() => {
      const res = this.api.scenes.value();
      if (res?.data) {
        this.rawScenes.set(res.data);
      }
    });

    const sub = this.liveUpdates.on('scene').subscribe((event: WsEvent) => {
      const now = new Date();
      this.lastUpdated.set(
        [
          now.getHours().toString().padStart(2, '0'),
          now.getMinutes().toString().padStart(2, '0'),
          now.getSeconds().toString().padStart(2, '0'),
        ].join(':')
      );

      if (event.operation === 'create' && event.data) {
        const created = event.data as Scene;
        this.rawScenes.update((list) =>
          list.some((s) => s.id === created.id) ? list : [...list, created]
        );
      } else if (event.operation === 'update' && event.data) {
        const updated = event.data as Scene;
        this.rawScenes.update((list) =>
          list.map((s) => (s.id === updated.id ? updated : s))
        );
      } else if (event.operation === 'delete' && event.resource_id) {
        this.rawScenes.update((list) =>
          list.filter((s) => s.id !== event.resource_id)
        );
      }
    });

    this.destroyRef.onDestroy(() => {
      sub.unsubscribe();
    });
  }

  getSceneAccent(index: number): string {
    return ACCENTS[index % ACCENTS.length];
  }

  openScene(scene: Scene): void {
    this.telemetry.track('scene.open', { scene: scene.name.toLowerCase() });
  }

  onFocus(scene: Scene): void {
    this.telemetry.track('launcher.focus', { scene: scene.name.toLowerCase() });
  }
}