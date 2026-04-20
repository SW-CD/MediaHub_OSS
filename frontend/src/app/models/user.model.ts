// Define the Permission structure based on the new concept
export interface Permission {
  database_name: string;
  can_view: boolean;
  can_create: boolean;
  can_edit: boolean;
  can_delete: boolean;
}

// Update the User interface to perfectly match the backend JSON
export interface User {
  id: number;          // Note: Changed to number based on your JSON ("id": 3)
  username: string;
  is_admin: boolean;
  permissions: Permission[]; 
  
  // Optional tracking fields (if your backend still returns them)
  created_at?: string;
  updated_at?: string;
}