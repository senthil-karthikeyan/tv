import { ApplicationConfig, provideBrowserGlobalErrorListeners } from '@angular/core';
import { provideRouter } from '@angular/router';
import { routes } from './app.routes';
import { provideClientHydration, withEventReplay } from '@angular/platform-browser';
import { provideHttpClient, withInterceptors } from '@angular/common/http';
import { apiBaseUrlInterceptor } from './core/http/api-base-url-interceptor';
import { apiKeyInterceptor } from './core/http/api-key-interceptor';
import { API_CONFIG } from './core/http/api-config';

export const appConfig: ApplicationConfig = {
  providers: [
    provideBrowserGlobalErrorListeners(),
    provideRouter(routes),
    provideClientHydration(withEventReplay()),
    provideHttpClient(withInterceptors([apiBaseUrlInterceptor, apiKeyInterceptor])),

    {
      provide: API_CONFIG,
      useValue: {
        baseUrl: 'https://tv-q7ok.onrender.com',
        wsUrl: 'https://tv-q7ok.onrender.com',
        apiKey: '07c47f7ca964081980cde93eb7cd288fad91bd34ac2558392b39d3f8815a9ba4',
      },
    },
  ],
};
