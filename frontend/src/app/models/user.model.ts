// Define the Permission structure based on the new concept
export interface Permission {
  database_id: string; // UPDATED: Replaced database_name with database_id
  can_view: boolean;
  can_create: boolean;
  can_edit: boolean;
  can_delete: boolean;
  can_admin: boolean;
}

// Update the User interface to perfectly match the backend JSON
export interface User {
  id: string; // Changed from number to string for ULID
  username: string;
  is_admin: boolean;
  is_service_account: boolean;
  permissions: Permission[]; 
  
  // Optional tracking fields (if your backend still returns them)
  created_at?: string;
  updated_at?: string;
}