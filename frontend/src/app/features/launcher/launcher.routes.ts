import { Routes } from '@angular/router';

export const LAUNCHER_ROUTES: Routes = [
  {
    path: '',
    loadComponent: () => import('./launcher').then((m) => m.Launcher),
  },
];
