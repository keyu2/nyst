import type { RouteAlert } from "./api";

type Alert = Partial<RouteAlert> & { cause?: number; mercury?: { alert_type?: string } };

export function isPlannedServiceAlert(alert: Alert = {}) {
  const alertType = String(alert.alertType || alert.mercury?.alert_type || "").trim();
  return alert.cause === 9 || alert.cause === 10 || /^planned\b/i.test(alertType) || /^no scheduled service$/i.test(alertType) || /\bstation notice\b/i.test(alertType) || /\b(?:saturday|sunday|weekend|holiday|special) schedule\b/i.test(alertType);
}

const delayPattern = /\b(delay|delays|delayed)\b/i;
const suspensionPattern = /\b(part(?:ially)? suspended|suspended|no (?:\[[^\]]+\]|\S+) service|service suspended)\b/i;

export function serviceAlertStatus(alert: Alert = {}) {
  if (alert.category === "planned" || isPlannedServiceAlert(alert)) return "planned";
  const alertType = String(alert.alertType || alert.mercury?.alert_type || "");
  if (delayPattern.test(alertType)) return "delay";
  if (suspensionPattern.test(alertType)) return "suspended";
  if (suspensionPattern.test(`${alert.header || ""} ${alert.description || ""}`)) return "suspended";
  return "delay";
}

export function serviceAlertTone(alert: Alert = {}) {
  const status = serviceAlertStatus(alert);
  if (status === "suspended") return "critical";
  if (status === "delay") return "warning";
  return "planned";
}

export function serviceAlertLabel(alert: Alert = {}, fallback = "Service alert") {
  return String(alert.alertType || fallback).replace(/\s*[–—-]\s*/g, " - ");
}
