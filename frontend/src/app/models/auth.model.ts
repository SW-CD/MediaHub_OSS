import { User } from './index';

export interface TokenResponse {
  access_token: string;
  refresh_token: string;
  user?: User;
}