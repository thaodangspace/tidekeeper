/**
 * Time helpers. All persisted/public instants are UTC (`Z`). Game-day keys
 * and lock instants use the configured IANA game timezone.

/** Returns a new Date at the current UTC instant. */
export function nowUTC(): Date {
  return new Date();
}

/** Formats a Date as a UTC day-key string in YYYY-MM-DD form. */
export function utcDayKey(date: Date): string {
  return date.toISOString().slice(0, 10);
}

/** Formats a Date as the local game day-key in YYYY-MM-DD form. */
export function dayKeyInTimezone(date: Date, timeZone: string): string {
  const parts = localParts(date, timeZone);
  return `${parts.year}-${String(parts.month).padStart(2, "0")}-${
    String(parts.day).padStart(2, "0")
  }`;
}

/** Returns the current UTC day-key. */
export function todayDayKey(): string {
  return utcDayKey(new Date());
}

/**
 * Computes the UTC instant at `minutesOfDay` (0-1439) in `timeZone` on the
 * day identified by `dayKey`. The returned value is always an absolute UTC
 * instant suitable for persistence and API responses.
 */
export function instantAtMinutesUtc(
  dayKey: string,
  minutesOfDay: number,
  timeZone = "UTC",
): Date {
  const [year, month, day] = dayKey.split("-").map(Number);
  const guess = new Date(Date.UTC(year, month - 1, day, 0, minutesOfDay));
  const actual = localParts(guess, timeZone);
  const actualAsUtc = Date.UTC(
    actual.year,
    actual.month - 1,
    actual.day,
    actual.hour,
    actual.minute,
  );
  const desiredAsUtc = Date.UTC(year, month - 1, day, 0, minutesOfDay);
  return new Date(guess.getTime() + desiredAsUtc - actualAsUtc);
}

function localParts(date: Date, timeZone: string): {
  year: number;
  month: number;
  day: number;
  hour: number;
  minute: number;
} {
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  }).formatToParts(date);
  const values = new Map(
    parts.filter((part) => part.type !== "literal").map((part) => [
      part.type,
      Number.parseInt(part.value, 10),
    ]),
  );
  return {
    year: values.get("year")!,
    month: values.get("month")!,
    day: values.get("day")!,
    hour: values.get("hour")!,
    minute: values.get("minute")!,
  };
}

/** Adds a number of milliseconds to a Date and returns a new Date. */
export function addMs(date: Date, ms: number): Date {
  return new Date(date.getTime() + ms);
}
