export interface ISpotifyAccessTokenResponse {
  access_token: string;
  token_type: 'Bearer';
  scope: string;
  expires_in: number;
  refresh_token: string;
}

export interface ISpotifyRenewAccessTokenResponse {
  access_token: string;
  token_type: 'Bearer';
  scope: string;
  expires_in: number;
}
