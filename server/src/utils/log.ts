/**
 * Structured logging.
 *
 * JSON line output (one object per line) to stdout, matching the Go server's
 * zerolog-style structured logging. A lightweight level filter is applied so
 * debug lines can be toggled without touching call sites.
 */

export type LogLevel = "debug" | "info" | "warn" | "error";

const LEVEL_ORDER: Record<LogLevel, number> = {
  debug: 0,
  info: 1,
  warn: 2,
  error: 3,
};

export interface LogFields {
  [key: string]: unknown;
}

export class Logger {
  #level: LogLevel;
  #output: Pick<Console, "log" | "error">;
  #baseFields: LogFields;

  constructor(
    level: LogLevel = "info",
    output: Pick<Console, "log" | "error"> = console,
    baseFields: LogFields = {},
  ) {
    this.#level = level;
    this.#output = output;
    this.#baseFields = baseFields;
  }

  setLevel(level: LogLevel): void {
    this.#level = level;
  }

  #write(level: LogLevel, message: string, fields: LogFields): void {
    if (LEVEL_ORDER[level] < LEVEL_ORDER[this.#level]) {
      return;
    }
    const line = JSON.stringify({
      time: new Date().toISOString(),
      level,
      message,
      ...this.#baseFields,
      ...fields,
    });
    if (level === "error") {
      this.#output.error(line);
    } else {
      this.#output.log(line);
    }
  }

  debug(message: string, fields: LogFields = {}): void {
    this.#write("debug", message, fields);
  }

  info(message: string, fields: LogFields = {}): void {
    this.#write("info", message, fields);
  }

  warn(message: string, fields: LogFields = {}): void {
    this.#write("warn", message, fields);
  }

  error(message: string, fields: LogFields = {}): void {
    this.#write("error", message, fields);
  }

  /** Returns a logger that prepends the given fields to every line. */
  child(fields: LogFields): Logger {
    return new Logger(this.#level, this.#output, {
      ...this.#baseFields,
      ...fields,
    });
  }
}

/** The global logger used across the application. */
export const logger = new Logger();

/** Configures the global logger level at startup. */
export function configureLogger(level: LogLevel): void {
  logger.setLevel(level);
}

/** Creates a configured logger with base fields applied to every line. */
export function createLogger(options: {
  level?: LogLevel;
  output?: Pick<Console, "log" | "error">;
  service?: string;
  version?: string;
}): Logger {
  const baseFields: LogFields = {};
  if (options.service !== undefined) {
    baseFields.service = options.service;
  }
  if (options.version !== undefined) {
    baseFields.version = options.version;
  }
  return new Logger(
    options.level ?? "info",
    options.output ?? console,
    baseFields,
  );
}
