export interface AvailableStylusResponse {
  styluses: AvailableStylus[];
}

export interface AvailableStylus {
  id: string;
  brand: string;
  model: string;
  type?: string;
  recommendedReplaceHours?: number;
  isVerified: boolean;
  userGeneratedId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface UserStylusesResponse {
  styluses: UserStylus[];
}

export interface UserStylus {
  id: string;
  userId: string;
  stylusId: string;
  stylus?: AvailableStylus;
  purchaseDate?: string;
  installDate?: string;
  hoursUsed?: number;
  notes?: string;
  isActive: boolean;
  isPrimary: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface CreateUserStylusRequest {
  stylusId: string;
  purchaseDate?: string;
  installDate?: string;
  hoursUsed?: number;
  notes?: string;
  isActive?: boolean;
  isPrimary?: boolean;
}

export interface CreateCustomStylusRequest {
  brand: string;
  model: string;
  type?: string;
  recommendedReplaceHours?: number;
}

export interface CreateCustomStylusResponse {
  stylus: AvailableStylus;
}

export interface UpdateUserStylusRequest {
  purchaseDate?: string;
  installDate?: string;
  hoursUsed?: number;
  notes?: string;
  isActive?: boolean;
  isPrimary?: boolean;
}
