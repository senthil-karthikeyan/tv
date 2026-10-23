import { Routes } from '@angular/router';

export const routes: Routes = [
  { path: '', pathMatch: 'full', redirectTo: 'launcher' },
  {
    path: 'launcher',
    loadChildren: () =>
      import('./features/launcher/launcher.routes').then((m) => m.LAUNCHER_ROUTES),
  },
  {
    path: 'dashboard',
    loadChildren: () =>
      import('./features/dashboard/dashboard.routes').then((m) => m.DASHBOARD_ROUTES),
  },
  {
    path: 'users',
    loadChildren: () =>
      import('./features/users/users.routes').then((m) => m.USERS_ROUTES),
  },
  {
    path: '**',
    loadComponent: () =>
      import('./features/dynamic-page/dynamic-page').then((m) => m.DynamicPage),
  },
];
