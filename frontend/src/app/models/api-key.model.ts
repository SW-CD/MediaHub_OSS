// frontend/src/app/models/api-key.model.ts

import { User } from './user.model';

export interface ApiKey {
  id: string; // ULID
  name: string;
  key_hint: string;
  token?: string; // Plaintext token secret, returned ONLY ONCE upon creation
  
  // Token Scopes (Filters)
  scope_view: boolean;
  scope_create: boolean;
  scope_edit: boolean;
  scope_delete: boolean;
  scope_admin: boolean;
  
  created_at: number; // millisecond timestamp
  expires_at: number | null; // millisecond timestamp
  last_used_at: number | null; // millisecond timestamp
  
  user?: Partial<User>; // optional embedded user details for global lists
}
