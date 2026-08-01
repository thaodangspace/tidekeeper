/** Build metadata equivalent to the Go buildinfo package. */

export interface BuildInfo {
  version: string;
  commit: string;
  buildTime: string;
}

export function buildInfo(
  lookup: (name: string) => string | undefined = (name) => Deno.env.get(name),
): BuildInfo {
  return {
    version: lookup("SERVICE_VERSION")?.trim() || "development",
    commit: lookup("GIT_COMMIT")?.trim() || "unknown",
    buildTime: lookup("BUILD_TIME")?.trim() || "unknown",
  };
}

export function buildInfoString(info = buildInfo()): string {
  return `tidekeepers-api version=${info.version} commit=${info.commit} built=${info.buildTime}`;
}
