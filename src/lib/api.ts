const baseURL = (import.meta.env.PUBLIC_API_BASE_URL ?? '').replace(/\/$/, '');

// PUBLIC_API_BASE_URL is compiled into the static bundle. Leave it empty when
// the static host reverse-proxies the API under the same origin.
export function apiURL(path: string): string {
  return `${baseURL}${path}`;
}

export function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  return fetch(apiURL(path), { ...init, credentials: 'include' });
}
