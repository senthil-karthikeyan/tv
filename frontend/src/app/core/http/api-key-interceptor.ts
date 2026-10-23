import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { API_CONFIG } from './api-config';

export const apiKeyInterceptor: HttpInterceptorFn = (req, next) => {
  const config = inject(API_CONFIG);

  const needsKey = ['POST', 'PUT', 'DELETE'].includes(req.method);

  if (needsKey && req.url.startsWith(config.baseUrl)) {
    req = req.clone({ setHeaders: { 'X-API-Key': config.apiKey } });
  }

  return next(req);
};