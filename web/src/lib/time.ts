const timeZone = "America/New_York";
const shortTime = new Intl.DateTimeFormat("en-US", { timeZone, hour: "numeric", minute: "2-digit" });
const clock = new Intl.DateTimeFormat("en-US", { timeZone, hour: "2-digit", minute: "2-digit", second: "2-digit", hourCycle: "h23" });
const offset = new Intl.DateTimeFormat("en-US", { timeZone, timeZoneName: "longOffset" });

function newYorkClockParts() {
  return Object.fromEntries(clock.formatToParts(new Date()).map((part) => [part.type, Number(part.value) || 0]));
}

export function currentNewYorkMinute() {
  const { hour, minute } = newYorkClockParts();
  return hour * 60 + minute;
}

export function currentNewYorkSecond() {
  const { hour, minute, second } = newYorkClockParts();
  return hour * 3600 + minute * 60 + second;
}

export function gtfsMinute(value: unknown) {
  const [hours, minutes] = String(value || "").split(":").map(Number);
  return Number.isFinite(hours) && Number.isFinite(minutes) ? hours * 60 + minutes : null;
}

export function gtfsSecond(value: unknown) {
  const [hours, minutes, seconds] = String(value || "").split(":").map(Number);
  return Number.isFinite(hours) && Number.isFinite(minutes)
    ? hours * 3600 + minutes * 60 + (Number.isFinite(seconds) ? seconds : 0)
    : null;
}

export function unixSeconds(value: unknown) {
  if (value == null) return undefined;
  if (typeof value === "object" && "toNumber" in value && typeof value.toNumber === "function") return value.toNumber();
  return Number(value);
}

export function tripIdStartSecond(tripId: string) {
  const encodedPart = String(tripId || "").split("_")[0];
  return /^\d+$/.test(encodedPart) ? Math.round(Number(encodedPart) * 0.6) : null;
}

export function tripStartTimestamp(serviceDate: string, startSecond: number | null) {
  if (!/^\d{8}$/.test(serviceDate || "") || startSecond === null) return null;
  const year = Number(serviceDate.slice(0, 4));
  const month = Number(serviceDate.slice(4, 6));
  const day = Number(serviceDate.slice(6, 8));
  const noon = Date.UTC(year, month - 1, day, 12);
  const zone = offset.formatToParts(new Date(noon)).find((part) => part.type === "timeZoneName")?.value;
  const match = zone?.match(/GMT([+-])(\d{2}):(\d{2})/);
  if (!match) return null;
  const offsetSeconds = (Number(match[2]) * 3600 + Number(match[3]) * 60) * (match[1] === "+" ? 1 : -1);
  return noon / 1000 - offsetSeconds - 12 * 3600 + startSecond;
}

export function formatArrivalCountdown(totalSeconds: number | null, showSeconds = false) {
  if (totalSeconds == null || !Number.isFinite(totalSeconds)) return "-";
  const remaining = Math.max(0, Math.floor(totalSeconds));
  const hours = Math.floor(remaining / 3600);
  const minutes = Math.floor((remaining % 3600) / 60);
  const seconds = remaining % 60;
  if (hours > 0) return `${hours}h ${minutes}m${showSeconds ? ` ${seconds}s` : ""}`;
  if (showSeconds) return `${Math.floor(remaining / 60)}m ${seconds}s`;
  const rounded = Math.round(remaining / 60);
  return rounded === 0 ? "0m" : `${rounded} min`;
}

export function formatNewYorkTime(timestamp: number | undefined) {
  return timestamp ? shortTime.format(new Date(timestamp * 1000)) : "-";
}
