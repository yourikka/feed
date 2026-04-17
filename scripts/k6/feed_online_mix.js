import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TOKEN = __ENV.TOKEN || "";
const TOKEN_POOL_SIZE = Number(__ENV.TOKEN_POOL_SIZE || 30);

export const options = {
  scenarios: {
    online_mix: {
      executor: "ramping-arrival-rate",
      startRate: 500,
      timeUnit: "1s",
      preAllocatedVUs: 200,
      maxVUs: 1500,
      stages: [
        { target: 1500, duration: "30s" },
        { target: 3000, duration: "30s" },
        { target: 3000, duration: "2m" },
        { target: 0, duration: "30s" },
      ],
      exec: "mixedTraffic",
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<300"],
  },
};

function buildUsername() {
  // 后端要求 3~20，仅允许字母数字下划线
  const ts = Date.now().toString(36).slice(-6);
  const rand = Math.floor(Math.random() * 1e6).toString(36).padStart(4, "0").slice(-4);
  return `k6_${ts}_${rand}`;
}

function authHeaders(token) {
  return token ? { headers: { Authorization: `Bearer ${token}` } } : {};
}

export function setup() {
  // 可选：外部传 TOKEN 时，直接复用，避免批量注册
  if (TOKEN) {
    const seed = http.get(`${BASE_URL}/douyin/feed/?sort=latest&limit=20`, authHeaders(TOKEN));
    const seedBody = seed.json() || {};
    const ids = (seedBody.video_list || []).map((v) => v.ID).filter(Boolean);
    return { tokens: [TOKEN], videoIds: ids };
  }

  const tokens = [];
  for (let i = 0; i < TOKEN_POOL_SIZE; i += 1) {
    const username = buildUsername();
    const password = "K6pass123!";

    const reg = http.post(
      `${BASE_URL}/douyin/user/register/`,
      JSON.stringify({ username, password }),
      { headers: { "Content-Type": "application/json" } },
    );
    const body = reg.json() || {};
    if (body.status_code === 0 && body.token) {
      tokens.push(body.token);
    }
  }

  if (tokens.length === 0) {
    throw new Error("setup failed: no valid token created");
  }

  const seed = http.get(`${BASE_URL}/douyin/feed/?sort=latest&limit=20`, authHeaders(tokens[0]));
  const seedBody = seed.json() || {};
  const ids = (seedBody.video_list || []).map((v) => v.ID).filter(Boolean);

  return { tokens, videoIds: ids };
}

function pickToken(tokens) {
  return tokens[Math.floor(Math.random() * tokens.length)];
}

function pickVideoId(videoIds) {
  if (!videoIds || videoIds.length === 0) return 0;
  return videoIds[Math.floor(Math.random() * videoIds.length)];
}

export function mixedTraffic(data) {
  const token = pickToken(data.tokens || []);
  const headers = authHeaders(token);
  const dice = Math.random();

  // 55% latest feed（读主流量）
  if (dice < 0.55) {
    const r = http.get(`${BASE_URL}/douyin/feed/?sort=latest&limit=10`, headers);
    check(r, { "latest feed 200": (res) => res.status === 200 });
    return;
  }

  // 30% hot feed（热点链路）
  if (dice < 0.85) {
    const r = http.get(`${BASE_URL}/douyin/feed/?sort=hot&limit=10`, headers);
    check(r, { "hot feed 200": (res) => res.status === 200 });
    return;
  }

  const videoId = pickVideoId(data.videoIds);
  if (!videoId) {
    const r = http.get(`${BASE_URL}/douyin/feed/?sort=latest&limit=10`, headers);
    check(r, { "fallback latest 200": (res) => res.status === 200 });
    return;
  }

  // 10% comment list
  if (dice < 0.95) {
    const r = http.get(`${BASE_URL}/douyin/comment/list/?video_id=${videoId}&limit=20`, headers);
    check(r, { "comment list 200": (res) => res.status === 200 });
    return;
  }

  // 5% like action（写路径）
  const r = http.post(`${BASE_URL}/douyin/like/action/?video_id=${videoId}`, null, headers);
  check(r, { "like action 200": (res) => res.status === 200 });
}
