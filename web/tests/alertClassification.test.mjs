import test from "node:test";
import assert from "node:assert/strict";
import { isPlannedServiceAlert, serviceAlertStatus } from "../src/lib/alertClassification.ts";

test("treats schedule announcements as planned service alerts", () => {
  for (const alertType of [
    "Saturday Schedule",
    "Sunday Schedule",
    "Weekend Schedule",
    "Holiday Schedule",
    "Special Schedule",
    "No Scheduled Service",
  ]) {
    assert.equal(isPlannedServiceAlert({ alertType }), true, alertType);
  }
});

test("does not classify live delays as planned service alerts", () => {
  assert.equal(isPlannedServiceAlert({ alertType: "Delays" }), false);
});

test("keeps an explicitly labeled delay orange even when details mention no service", () => {
  assert.equal(serviceAlertStatus({
    alertType: "Delays",
    category: "happening",
    description: "Some trains are delayed because another service has no service.",
  }), "delay");
});

test("classifies explicit and descriptive suspensions as suspended", () => {
  assert.equal(serviceAlertStatus({
    alertType: "Part Suspended",
    category: "happening",
  }), "suspended");
  assert.equal(serviceAlertStatus({
    alertType: "Service Change",
    category: "happening",
    description: "No [6] service between these stations.",
  }), "suspended");
});

