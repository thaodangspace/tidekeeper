import { normalizeClientIp } from "../auth/identity.ts";

/**
 * Resolves the effective connection address once at the Deno.serve boundary.
 * Forwarding metadata is considered only when TRUSTED_PROXY is explicitly on;
 * application handlers never inspect client-supplied forwarding headers.
 */
export function resolveClientIp(
  request: Request,
  remoteAddr: { hostname?: string },
  proxy: { trusted: boolean; header: string },
): string | null {
  if (proxy.trusted) {
    const forwarded = request.headers.get(proxy.header);
    const first = forwarded?.split(",")[0]?.trim() ?? "";
    const normalized = normalizeClientIp(first);
    if (normalized) return normalized;
  }
  return normalizeClientIp(remoteAddr.hostname ?? "");
}
