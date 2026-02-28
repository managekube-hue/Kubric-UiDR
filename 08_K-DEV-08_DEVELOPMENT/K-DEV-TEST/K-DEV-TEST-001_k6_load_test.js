// ─────────────────────────────────────────────────────────────────────────────
// Kubric-UiDR — K6 Load Test Suite
// Validates API performance under smoke, load, stress, and spike scenarios.
// Usage:
//   k6 run --env SCENARIO=smoke  K-DEV-TEST-001_k6_load_test.js
//   k6 run --env SCENARIO=load   K-DEV-TEST-001_k6_load_test.js
//   k6 run --env SCENARIO=stress K-DEV-TEST-001_k6_load_test.js
//   k6 run --env SCENARIO=spike  K-DEV-TEST-001_k6_load_test.js
// ─────────────────────────────────────────────────────────────────────────────

import http from "k6/http";
import { check, sleep, group } from "k6";
import { Rate, Trend, Counter } from "k6/metrics";

// ── Custom metrics ───────────────────────────────────────────────────────────
const healthzDuration    = new Trend("kubric_healthz_duration", true);
const alertsDuration     = new Trend("kubric_alerts_duration", true);
const scanDuration       = new Trend("kubric_scan_duration", true);
const complianceDuration = new Trend("kubric_compliance_duration", true);
const topologyDuration   = new Trend("kubric_topology_duration", true);
const errorRate          = new Rate("kubric_errors");
const requestCount       = new Counter("kubric_requests");

// ── Configuration ────────────────────────────────────────────────────────────
const BASE_URL   = __ENV.BASE_URL   || "http://localhost:8080";
const API_TOKEN  = __ENV.API_TOKEN  || "k6-load-test-token";
const SCENARIO   = __ENV.SCENARIO   || "smoke";

const scenarios = {
  smoke: {
    executor: "constant-vus",
    vus: 1,
    duration: "30s",
    tags: { scenario: "smoke" },
  },
  load: {
    executor: "ramping-vus",
    startVUs: 0,
    stages: [
      { duration: "1m",  target: 25  },
      { duration: "3m",  target: 50  },
      { duration: "1m",  target: 0   },
    ],
    tags: { scenario: "load" },
  },
  stress: {
    executor: "ramping-vus",
    startVUs: 0,
    stages: [
      { duration: "2m",  target: 50  },
      { duration: "5m",  target: 200 },
      { duration: "2m",  target: 200 },
      { duration: "1m",  target: 0   },
    ],
    tags: { scenario: "stress" },
  },
  spike: {
    executor: "ramping-vus",
    startVUs: 0,
    stages: [
      { duration: "10s", target: 500 },
      { duration: "1m",  target: 500 },
      { duration: "10s", target: 0   },
    ],
    tags: { scenario: "spike" },
  },
};

export const options = {
  scenarios: {
    default: scenarios[SCENARIO] || scenarios.smoke,
  },
  thresholds: {
    http_req_duration:          ["p(95)<500", "p(99)<1000"],
    http_req_failed:            ["rate<0.01"],
    kubric_healthz_duration:    ["p(95)<100"],
    kubric_alerts_duration:     ["p(95)<400"],
    kubric_scan_duration:       ["p(95)<800"],
    kubric_compliance_duration: ["p(95)<500"],
    kubric_topology_duration:   ["p(95)<600"],
    kubric_errors:              ["rate<0.02"],
  },
  noConnectionReuse: false,
  userAgent: "KubricK6LoadTest/1.0",
};

// ── Helpers ──────────────────────────────────────────────────────────────────
const headers = {
  "Content-Type":  "application/json",
  "Authorization": `Bearer ${API_TOKEN}`,
  "X-Kubric-Module": "k6-load-test",
};

function checkResponse(res, name, customMetric) {
  const ok = check(res, {
    [`${name}: status 2xx`]:     (r) => r.status >= 200 && r.status < 300,
    [`${name}: duration < 1s`]:  (r) => r.timings.duration < 1000,
    [`${name}: body not empty`]: (r) => r.body && r.body.length > 0,
  });
  customMetric.add(res.timings.duration);
  requestCount.add(1);
  if (!ok) {
    errorRate.add(1);
  } else {
    errorRate.add(0);
  }
}

// ── Main test function ───────────────────────────────────────────────────────
export default function () {
  // ── Health check ─────────────────────────────────────────────────────────
  group("Health Check", () => {
    const res = http.get(`${BASE_URL}/healthz`, {
      headers,
      tags: { endpoint: "healthz", module: "platform" },
    });
    checkResponse(res, "healthz", healthzDuration);
  });

  sleep(0.5);

  // ── Alerts listing ───────────────────────────────────────────────────────
  group("List Alerts", () => {
    const res = http.get(`${BASE_URL}/api/v1/alerts?page=1&limit=25&severity=high`, {
      headers,
      tags: { endpoint: "alerts", module: "soc" },
    });
    checkResponse(res, "alerts", alertsDuration);
  });

  sleep(0.5);

  // ── Vulnerability scan trigger ───────────────────────────────────────────
  group("Trigger Scan", () => {
    const payload = JSON.stringify({
      target:    "192.168.1.0/24",
      scan_type: "quick",
      profile:   "default",
      tags:      ["k6-test", "automated"],
    });
    const res = http.post(`${BASE_URL}/api/v1/scan`, payload, {
      headers,
      tags: { endpoint: "scan", module: "vdr" },
    });
    checkResponse(res, "scan", scanDuration);
  });

  sleep(0.5);

  // ── Compliance frameworks ────────────────────────────────────────────────
  group("Compliance Frameworks", () => {
    const res = http.get(`${BASE_URL}/api/v1/compliance/frameworks`, {
      headers,
      tags: { endpoint: "compliance", module: "grc" },
    });
    checkResponse(res, "compliance", complianceDuration);
  });

  sleep(0.5);

  // ── Topology graph ──────────────────────────────────────────────────────
  group("Topology Graph", () => {
    const res = http.get(`${BASE_URL}/api/v1/topology/graph?depth=2&include_edges=true`, {
      headers,
      tags: { endpoint: "topology", module: "noc" },
    });
    checkResponse(res, "topology", topologyDuration);
  });

  sleep(1);
}

// ── Lifecycle hooks ──────────────────────────────────────────────────────────
export function setup() {
  const res = http.get(`${BASE_URL}/healthz`);
  if (res.status !== 200) {
    throw new Error(
      `API not reachable at ${BASE_URL}/healthz — got status ${res.status}`
    );
  }
  console.log(`Kubric API reachable. Running scenario: ${SCENARIO}`);
  return { startTime: new Date().toISOString() };
}

export function teardown(data) {
  console.log(`Load test completed. Started at: ${data.startTime}`);
}
