import { HttpInterceptorFn } from '@angular/common/http';
import { inject } from '@angular/core';
import { API_CONFIG } from './api-config';

export const apiBaseUrlInterceptor: HttpInterceptorFn = (req, next) => {
  const config = inject(API_CONFIG);

  if (req.url.startsWith('/api')) {
    req = req.clone({ url: config.baseUrl + req.url });
  }

  return next(req);
};