import { Component, computed, DestroyRef, effect, inject, signal } from '@angular/core';
import { FormControl, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
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

@Component({
  imports: [ReactiveFormsModule, FocusGroup, FocusItem],
  selector: 'app-users',
  templateUrl: './users.html',
})
export class Users {
  private api = inject(UsersApi);
  private liveUpdates = inject(LiveUpdates);
  private telemetry = inject(Telemetry);
  private router = inject(Router);
  private destroyRef = inject(DestroyRef);

  readonly usersResource = this.api.users;
  private rawUsers = signal<User[]>([]);
  readonly searchQuery = signal('');
  readonly editingId = signal<string | null>(null);
  readonly deletingId = signal<string | null>(null);
  readonly errorMessage = signal<string | null>(null);
  readonly isSubmitting = signal(false);

  readonly userForm = new FormGroup<UserForm>({
    name: new FormControl('', { nonNullable: true, validators: [Validators.required] }),
    email: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.email] }),
    role: new FormControl<UserRole>('viewer', { nonNullable: true, validators: [Validators.required] }),
    status: new FormControl<UserStatus>('active', { nonNullable: true, validators: [Validators.required] }),
  });

  readonly filteredUsers = computed(() => {
    const query = this.searchQuery().trim().toLowerCase();
    const list = this.rawUsers();
    if (!query) {
      return list;
    }
    return list.filter(
      (u) =>
        u.name.toLowerCase().includes(query) ||
        u.email.toLowerCase().includes(query)
    );
  });

  constructor() {
    this.telemetry.track('scene.open', { scene: 'users' });

    effect(() => {
      const res = this.api.users.value();
      if (res?.data) {
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
    this.router.navigateByUrl('/launcher');
  }

  onSearchChange(event: Event): void {
    const target = event.target as HTMLInputElement | null;
    this.searchQuery.set(target?.value ?? '');
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
  }

  cancelEdit(): void {
    this.editingId.set(null);
    this.errorMessage.set(null);
    this.userForm.reset({
      name: '',
      email: '',
      role: 'viewer',
      status: 'active',
    });
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
          this.cancelEdit();
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
          this.cancelEdit();
          this.isSubmitting.set(false);
        },
        error: (err: unknown) => {
          this.handleHttpError(err);
          this.isSubmitting.set(false);
        },
      });
    }
  }

  confirmDelete(id: string): void {
    this.deletingId.set(id);
  }

  cancelDelete(): void {
    this.deletingId.set(null);
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
