export type Preferences = {
  theme: "system" | "dark" | "light";
  showSeconds: boolean;
  tripProgressUnit: "minutes" | "stops";
  lineView: "overview" | "focus";
  mobileLineLayout: "grouped" | "rows";
  stationView: "standard" | "board";
  boardArrivalInversion: boolean;
  mapEnabled: boolean;
};
export const defaultPreferences: Preferences = { theme:"system", showSeconds:false, tripProgressUnit:"minutes", lineView:"overview", mobileLineLayout:"rows", stationView:"standard", boardArrivalInversion:true, mapEnabled:false };
const key = "subway-preferences";
export function readPreferences(): Preferences {
  try {
    const value = JSON.parse(localStorage.getItem(key) || "{}");
    const theme = localStorage.getItem("subway-theme");
    return {
      theme: theme === "dark" || theme === "light" ? theme : "system",
      showSeconds: value?.showSeconds === true,
      tripProgressUnit: value?.tripProgressUnit === "stops" ? "stops" : "minutes",
      lineView: value?.lineView === "focus" ? "focus" : "overview",
      mobileLineLayout: value?.mobileLineLayout === "grouped" ? "grouped" : "rows",
      stationView: value?.stationView === "board" ? "board" : "standard",
      boardArrivalInversion: value?.boardArrivalInversion !== false,
      mapEnabled: value?.mapEnabled === true,
    };
  } catch { return { ...defaultPreferences }; }
}
export function writePreferences(value: Preferences) {
  try {
    const { theme, ...preferences } = value;
    localStorage.setItem(key, JSON.stringify(preferences));
    if (theme === "system") localStorage.removeItem("subway-theme");
    else localStorage.setItem("subway-theme", theme);
  } catch { }
  applyTheme(value.theme);
  window.dispatchEvent(new CustomEvent("subway-preferences"));
  window.dispatchEvent(new CustomEvent("subway-preferences-change"));
  window.dispatchEvent(new CustomEvent("subway-theme-change"));
}
export function applyTheme(theme: Preferences["theme"]) {
  if (theme === "system") delete document.documentElement.dataset.theme;
  else document.documentElement.dataset.theme = theme;
}
export function watchPreferences(update: (value: Preferences) => void) {
  const sync = () => {
    const value = readPreferences();
    applyTheme(value.theme);
    update(value);
  };
  const events = ["storage", "subway-preferences-change", "subway-theme-change"];
  events.forEach((event) => window.addEventListener(event, sync));
  const scheme=window.matchMedia("(prefers-color-scheme: dark)");
  scheme.addEventListener("change",sync);
  return () => {
    events.forEach((event) => window.removeEventListener(event, sync));
    scheme.removeEventListener("change",sync);
  };
}
export function effectiveTheme(theme: Preferences["theme"]): "dark" | "light" {
  return theme==="system"?(window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"):theme;
}
