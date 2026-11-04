let stream: EventSource | null = null;
let connected = false;
let opened = false;
const listeners = new Set<() => void>();

export function watchLiveData(refresh: () => void, fallbackInterval = 15_000) {
  listeners.add(refresh);
  if (!stream && typeof EventSource !== "undefined") {
    stream = new EventSource("/api/events");
    stream.onopen = () => {
      connected = true;
      if (opened) listeners.forEach((listener) => listener());
      opened = true;
    };
    stream.onerror = () => connected = false;
    stream.addEventListener("refresh", () => listeners.forEach((listener) => listener()));
  }
  const fallback = window.setInterval(() => {
    if (!connected) refresh();
  }, fallbackInterval);
  return () => {
    listeners.delete(refresh);
    if (!listeners.size) {
      stream?.close();
      stream = null;
      connected = false;
      opened = false;
    }
    clearInterval(fallback);
  };
}
