import { Component, inject } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { LiveUpdates } from './core/realtime/live-updates';
import { Telemetry } from './core/telemetry/telemetry';

@Component({
  imports: [RouterOutlet],
  selector: 'app-root',
  templateUrl: './app.html',
})
export class App {
  private liveUpdates = inject(LiveUpdates);
  private telemetry = inject(Telemetry);
}
