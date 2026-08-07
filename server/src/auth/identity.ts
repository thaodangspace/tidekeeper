/** Canonical guest identity helpers. Raw client IPs never leave this module. */

import { base64UrlEncodeBytes } from "../utils/crypto.ts";
import { badRequest } from "../utils/errors.ts";

const MAX_DISPLAY_NAME_LENGTH = 32;
const MIN_DISPLAY_NAME_LENGTH = 2;

/** Normalizes a display name to the stable snake_case username used for identity. */
export function normalizeUsername(displayName: string): string {
  if (/\p{Cc}/u.test(displayName)) {
    throw badRequest("Enter a name between 2 and 32 characters.", {
      code: "INVALID_PLAYER_NAME",
    });
  }
  const trimmed = displayName.trim();
  const length = Array.from(trimmed).length;
  if (
    length < MIN_DISPLAY_NAME_LENGTH ||
    length > MAX_DISPLAY_NAME_LENGTH ||
    /\p{Cc}/u.test(trimmed)
  ) {
    throw badRequest("Enter a name between 2 and 32 characters.", {
      code: "INVALID_PLAYER_NAME",
    });
  }

  const normalized = trimmed.normalize("NFKC").toLowerCase()
    // Unicode punctuation and separators become word boundaries.
    .replace(/[\p{White_Space}\p{P}\p{S}_]+/gu, "_")
    .replace(/^_+|_+$/g, "");
  if (normalized === "" || /\p{Cc}/u.test(normalized)) {
    throw badRequest("Enter a name between 2 and 32 characters.", {
      code: "INVALID_PLAYER_NAME",
    });
  }
  return normalized;
}

/** Returns canonical textual IPv4/IPv6, or null for unavailable/invalid input. */
export function normalizeClientIp(value: string): string | null {
  let input = value.trim();
  if (input === "") return null;
  if (input.startsWith("[") && input.endsWith("]")) {
    input = input.slice(1, -1);
  }
  // Proxy metadata may include an IPv4 port, but never accept an ambiguous
  // IPv6 value with a port here.
  if (input.includes(".") && input.split(":").length === 2) {
    const colon = input.lastIndexOf(":");
    if (/^\d+$/.test(input.slice(colon + 1))) input = input.slice(0, colon);
  }
  input = input.split("%")[0]!; // Ignore IPv6 zone identifiers.
  return normalizeIpv4(input) ?? normalizeIpv6(input);
}

function normalizeIpv4(value: string): string | null {
  const parts = value.split(".");
  if (parts.length !== 4 || parts.some((part) => !/^\d{1,3}$/.test(part))) {
    return null;
  }
  const numbers = parts.map(Number);
  if (numbers.some((part) => part > 255)) return null;
  return numbers.join(".");
}

function normalizeIpv6(value: string): string | null {
  if (!value.includes(":")) return null;
  const halves = value.split("::");
  if (halves.length > 2) return null;
  const left = parseIpv6Groups(halves[0] ?? "");
  const right = halves.length === 2 ? parseIpv6Groups(halves[1] ?? "") : [];
  if (left === null || right === null) return null;
  const missing = 8 - left.length - right.length;
  if (halves.length === 1 && missing !== 0) return null;
  if (halves.length === 2 && missing < 1) return null;
  const groups = [...left, ...Array(missing).fill(0), ...right];
  const rendered = groups.map((group) => group.toString(16));
  let bestStart = -1;
  let bestLength = 0;
  for (let i = 0; i < rendered.length;) {
    if (rendered[i] !== "0") {
      i++;
      continue;
    }
    const start = i;
    while (i < rendered.length && rendered[i] === "0") i++;
    if (i - start > bestLength && i - start >= 2) {
      bestStart = start;
      bestLength = i - start;
    }
  }
  if (bestStart >= 0) {
    const before = rendered.slice(0, bestStart).join(":");
    const after = rendered.slice(bestStart + bestLength).join(":");
    return before === ""
      ? `::${after}`
      : after === ""
      ? `${before}::`
      : `${before}::${after}`;
  }
  return rendered.join(":");
}

function parseIpv6Groups(value: string): number[] | null {
  if (value === "") return [];
  const parts = value.split(":");
  const groups: number[] = [];
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i]!;
    if (part.includes(".")) {
      const ipv4 = normalizeIpv4(part);
      if (!ipv4 || i !== parts.length - 1) return null;
      const octets = ipv4.split(".").map(Number);
      groups.push((octets[0]! << 8) | octets[1]!);
      groups.push((octets[2]! << 8) | octets[3]!);
    } else if (!/^[0-9a-fA-F]{1,4}$/.test(part)) {
      return null;
    } else {
      groups.push(Number.parseInt(part, 16));
    }
  }
  return groups.length <= 8 ? groups : null;
}

/** Derives the opaque deterministic player id from username and canonical IP. */
export async function derivePlayerId(
  username: string,
  clientIp: string,
  secret: string,
): Promise<string> {
  const key = await crypto.subtle.importKey(
    "raw",
    new TextEncoder().encode(secret),
    { name: "HMAC", hash: "SHA-256" },
    false,
    ["sign"],
  );
  const digest = await crypto.subtle.sign(
    "HMAC",
    key,
    new TextEncoder().encode(`${username}|${clientIp}`),
  );
  return `ply_${base64UrlEncodeBytes(new Uint8Array(digest)).slice(0, 24)}`;
}
