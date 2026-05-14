import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TOKEN = __ENV.TOKEN || "";

export const options = {
  scenarios: {
    read_only: {
      executor: "ramping-arrival-rate",
      startRate: 800,
      timeUnit: "1s",
      preAllocatedVUs: 200,
      maxVUs: 1600,
      stages: [
        { target: 2000, duration: "30s" },
        { target: 3500, duration: "30s" },
        { target: 3500, duration: "2m" },
        { target: 0, duration: "30s" },
      ],
      exec: "readTraffic",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<450"],
  },
};

function authHeaders(token) {
  return token ? { headers: { Authorization: `Bearer ${token}` } } : {};
}

export function readTraffic() {
  const headers = authHeaders(TOKEN);
  const dice = Math.random();

  if (dice < 0.7) {
    const r = http.get(`${BASE_URL}/douyin/feed/?sort=latest&limit=10`, headers);
    check(r, { "latest feed 200": (res) => res.status === 200 });
    return;
  }

  const r = http.get(`${BASE_URL}/douyin/feed/?sort=hot&limit=10`, headers);
  check(r, { "hot feed 200": (res) => res.status === 200 });
}
