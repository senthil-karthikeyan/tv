import { inject, Injectable } from '@angular/core';
import { HttpClient, httpResource } from '@angular/common/http';
import { map, Observable } from 'rxjs';

export type UserRole = 'admin' | 'editor' | 'viewer';
export type UserStatus = 'active' | 'inactive' | 'suspended';

export interface User {
  id: string;
  name: string;
  email: string;
  role: UserRole;
  status: UserStatus;
  avatar?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateUserRequest {
  name: string;
  email: string;
  role: UserRole;
  status: UserStatus;
  avatar?: string;
}

export interface UpdateUserRequest {
  name?: string;
  email?: string;
  role?: UserRole;
  status?: UserStatus;
  avatar?: string;
}

export interface UsersResponse {
  data: User[];
  meta: { total: number };
}

export interface UserResponse {
  data: User;
}

@Injectable({ providedIn: 'root' })
export class UsersApi {
  private http = inject(HttpClient);

  users = httpResource<UsersResponse>(() => '/api/v1/users');

  create(request: CreateUserRequest): Observable<User> {
    return this.http
      .post<UserResponse>('/api/v1/users', request)
      .pipe(map((res) => res.data));
  }

  update(id: string, request: UpdateUserRequest): Observable<User> {
    return this.http
      .put<UserResponse>(`/api/v1/users/${id}`, request)
      .pipe(map((res) => res.data));
  }

  remove(id: string): Observable<void> {
    return this.http.delete<void>(`/api/v1/users/${id}`);
  }
}
