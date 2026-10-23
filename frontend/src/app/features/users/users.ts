import { Component, computed, DestroyRef, effect, inject, signal } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router, RouterLink } from '@angular/router';
import { HttpErrorResponse } from '@angular/common/http';
import { CreateUserRequest, UpdateUserRequest, User, UserRole, UserStatus, UsersApi } from './users-api';
import { LiveUpdates, WsEvent } from '../../core/realtime/live-updates';
import { Telemetry } from '../../core/telemetry/telemetry';
import { FocusGroup, FocusItem } from '../../shared/focus/focus-navigation';

interface UserForm {
  name: FormControl<string>;
  email: FormControl<string>;
  role: FormControl<UserRole>;
  status: FormControl<UserStatus>;
}

interface MockUserItem {
  id: string;
  name: string;
  role: string;
  city: string;
  status: string;
  chip: 'ok' | 'warn' | 'off';
  lastSeen: string;
}

@Component({
  imports: [ReactiveFormsModule, FocusGroup, FocusItem, RouterLink],
  selector: 'app-users',
  templateUrl: './users.html',
})
export class Users {
  private api = inject(UsersApi);
  private liveUpdates = inject(LiveUpdates);
  readonly telemetry = inject(Telemetry);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);

  readonly usersResource = this.api.users;
  readonly connectionStatus = this.liveUpdates.status;
  private rawUsers = signal<User[]>([]);
  readonly searchQuery = signal('');
  readonly editingId = signal<string | null>(null);
  readonly deletingId = signal<string | null>(null);
  readonly errorMessage = signal<string | null>(null);
  readonly isSubmitting = signal(false);
  readonly showAddModal = signal(false);
  readonly selectedUserId = signal<string | null>('mock-2');

  readonly demoUsers = signal<MockUserItem[]>([
    { id: 'mock-1', name: 'Asha Raman', role: 'Admin', city: 'Chennai', status: 'Active', chip: 'ok', lastSeen: 'Now' },
    { id: 'mock-2', name: 'Karthik S', role: 'Editor', city: 'Bengaluru', status: 'Active', chip: 'ok', lastSeen: '2 min ago' },
    { id: 'mock-3', name: 'Meera Nair', role: 'Viewer', city: 'Kochi', status: 'Invited', chip: 'warn', lastSeen: 'Pending' },
    { id: 'mock-4', name: 'Imran Khan', role: 'Editor', city: 'Hyderabad', status: 'Active', chip: 'ok', lastSeen: '1 hr ago' },
    { id: 'mock-5', name: 'Divya Menon', role: 'Viewer', city: 'Chennai', status: 'Suspended', chip: 'off', lastSeen: '3 days ago' },
  ]);

  readonly userForm = new FormGroup<UserForm>({
    name: new FormControl('', { nonNullable: true, validators: [Validators.required] }),
    email: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.email] }),
    role: new FormControl<UserRole>('viewer', { nonNullable: true, validators: [Validators.required] }),
    status: new FormControl<UserStatus>('active', { nonNullable: true, validators: [Validators.required] }),
  });

  readonly displayUsers = computed(() => {
    const apiList = this.rawUsers();
    const query = this.searchQuery().trim().toLowerCase();

    const merged: MockUserItem[] = apiList.length > 0
      ? apiList.map((u, i) => ({
          id: u.id,
          name: u.name,
          role: u.role.charAt(0).toUpperCase() + u.role.slice(1),
          city: (u as { city?: string }).city || (['Chennai', 'Bengaluru', 'Kochi', 'Hyderabad'][i % 4]),
          status: u.status.charAt(0).toUpperCase() + u.status.slice(1),
          chip: u.status === 'active' ? 'ok' : ((u.status as string) === 'invited' ? 'warn' : 'off'),
          lastSeen: 'Now',
        }))
      : this.demoUsers();

    if (!query) {
      return merged;
    }

    return merged.filter(
      (u) =>
        u.name.toLowerCase().includes(query) ||
        u.city.toLowerCase().includes(query) ||
        u.role.toLowerCase().includes(query)
    );
  });

  constructor() {
    this.telemetry.track('scene.open', { scene: 'users' });

    effect(() => {
      const res = this.api.users.value();
      if (res?.data && res.data.length > 0) {
        this.rawUsers.set(res.data);
      }
    });

    const sub = this.liveUpdates.on('user').subscribe((event: WsEvent) => {
      if (event.operation === 'create' && event.data) {
        const created = event.data as User;
        this.rawUsers.update((list) =>
          list.some((u) => u.id === created.id) ? list : [...list, created]
        );
      } else if (event.operation === 'update' && event.data) {
        const updated = event.data as User;
        this.rawUsers.update((list) =>
          list.map((u) => (u.id === updated.id ? updated : u))
        );
      } else if (event.operation === 'delete' && event.resource_id) {
        this.rawUsers.update((list) =>
          list.filter((u) => u.id !== event.resource_id)
        );
      }
    });

    this.destroyRef.onDestroy(() => {
      sub.unsubscribe();
    });
  }

  goBack(): void {
    this.telemetry.track('scene.back', { scene: 'users' });
    this.router.navigateByUrl('/launcher');
  }

  onSearchChange(event: Event): void {
    const target = event.target as HTMLInputElement | null;
    this.searchQuery.set(target?.value ?? '');
  }

  selectUser(user: MockUserItem): void {
    this.selectedUserId.set(user.id);
    this.telemetry.track('user.select', { user: user.name });
  }

  openAddUser(): void {
    this.editingId.set(null);
    this.errorMessage.set(null);
    this.userForm.reset({
      name: '',
      email: '',
      role: 'viewer',
      status: 'active',
    });
    this.showAddModal.set(true);
  }

  closeAddUser(): void {
    this.showAddModal.set(false);
    this.editingId.set(null);
    this.errorMessage.set(null);
  }

  startEdit(user: User): void {
    this.telemetry.track('user.select', { user: user.name });
    this.editingId.set(user.id);
    this.errorMessage.set(null);
    this.userForm.setValue({
      name: user.name,
      email: user.email,
      role: user.role,
      status: user.status,
    });
    this.showAddModal.set(true);
  }

  onSubmit(): void {
    if (this.userForm.invalid || this.isSubmitting()) {
      return;
    }

    this.isSubmitting.set(true);
    this.errorMessage.set(null);
    const formValue = this.userForm.getRawValue();
    const editId = this.editingId();

    if (editId) {
      const req: UpdateUserRequest = {
        name: formValue.name,
        email: formValue.email,
        role: formValue.role,
        status: formValue.status,
      };

      this.api.update(editId, req).subscribe({
        next: (updated) => {
          this.rawUsers.update((list) =>
            list.map((u) => (u.id === updated.id ? updated : u))
          );
          this.closeAddUser();
          this.isSubmitting.set(false);
        },
        error: (err: unknown) => {
          this.handleHttpError(err);
          this.isSubmitting.set(false);
        },
      });
    } else {
      const req: CreateUserRequest = {
        name: formValue.name,
        email: formValue.email,
        role: formValue.role,
        status: formValue.status,
      };

      this.api.create(req).subscribe({
        next: (created) => {
          this.rawUsers.update((list) =>
            list.some((u) => u.id === created.id) ? list : [...list, created]
          );
          this.closeAddUser();
          this.isSubmitting.set(false);
        },
        error: (err: unknown) => {
          this.handleHttpError(err);
          this.isSubmitting.set(false);
        },
      });
    }
  }

  deleteUser(id: string): void {
    this.api.remove(id).subscribe({
      next: () => {
        this.rawUsers.update((list) => list.filter((u) => u.id !== id));
        this.deletingId.set(null);
      },
      error: (err: unknown) => {
        this.handleHttpError(err);
        this.deletingId.set(null);
      },
    });
  }

  private handleHttpError(err: unknown): void {
    if (err instanceof HttpErrorResponse) {
      if (err.status === 409 && err.error?.error?.code === 'DUPLICATE_EMAIL') {
        this.errorMessage.set(err.error?.error?.message ?? 'Email already in use');
        return;
      }
      if (err.error?.error?.message) {
        this.errorMessage.set(err.error.error.message as string);
        return;
      }
    }
    this.errorMessage.set('An error occurred. Please try again.');
  }
}
