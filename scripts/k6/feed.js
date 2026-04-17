import http from "k6/http";
import { check, sleep } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TEST_MODE = __ENV.TEST_MODE || "business";
const TOKEN = __ENV.TOKEN || "";

function buildOptions() {
  if (TEST_MODE === "extreme") {
    // 极限吞吐模式：单接口、无 sleep，按到达率阶梯加压
    return {
      scenarios: {
        extreme_latest: {
          executor: "ramping-arrival-rate",
          startRate: 200,
          timeUnit: "1s",
          preAllocatedVUs: 100,
          maxVUs: 1200,
          stages: [
            { target: 500, duration: "30s" },
            { target: 1000, duration: "30s" },
            { target: 2000, duration: "30s" },
            { target: 3000, duration: "30s" },
            { target: 0, duration: "20s" },
          ],
        },
      },
      thresholds: {
        http_req_failed: ["rate<0.01"],
      },
    };
  }

  // 业务混合模式：读 latest/hot + like
  return {
    stages: [
      { duration: "30s", target: 20 },
      { duration: "1m", target: 20 },
      { duration: "30s", target: 50 },
      { duration: "1m", target: 50 },
      { duration: "30s", target: 0 },
    ],
    thresholds: {
      http_req_failed: ["rate<0.01"],
      http_req_duration: ["p(95)<300"],
    },
  };
}

export const options = buildOptions();

function buildUsername() {
  // 后端限制用户名 3~20 且仅允许字母数字下划线
  const ts = Date.now().toString(36).slice(-6);
  const rand = Math.floor(Math.random() * 1e6).toString(36).padStart(4, "0").slice(-4);
  return `k6_${ts}_${rand}`; // 长度约 14
}

export function setup() {
  if (TEST_MODE === "extreme") {
    return {};
  }

  const password = "K6pass123!";
  const headers = { headers: { "Content-Type": "application/json" } };

  // 避免偶发重名，注册失败时重试几次
  for (let i = 0; i < 5; i += 1) {
    const username = buildUsername();
    const reg = http.post(
      `${BASE_URL}/douyin/user/register/`,
      JSON.stringify({ username, password }),
      headers,
    );
    const body = reg.json();
    if (body && body.status_code === 0 && body.token) {
      return { token: body.token };
    }
  }

  throw new Error("register failed after retries");
}

export default function (data) {
  if (TEST_MODE === "extreme") {
    const headers = TOKEN ? { headers: { Authorization: `Bearer ${TOKEN}` } } : undefined;
    const res = http.get(`${BASE_URL}/douyin/feed/?sort=latest&limit=10`, headers);
    check(res, { "extreme latest feed ok": (r) => r.status === 200 });
    return;
  }

  const authHeaders = { headers: { Authorization: `Bearer ${data.token}` } };

  const latest = http.get(`${BASE_URL}/douyin/feed/?sort=latest&limit=10`, authHeaders);
  check(latest, { "latest feed ok": (r) => r.status === 200 });

  const hot = http.get(`${BASE_URL}/douyin/feed/?sort=hot&limit=10`, authHeaders);
  check(hot, { "hot feed ok": (r) => r.status === 200 });

  let videoId = 0;
  try {
    const latestBody = latest.json();
    if (latestBody?.video_list?.length > 0) {
      videoId = latestBody.video_list[0].ID;
    }
  } catch (_) {}

  if (videoId) {
    const like = http.post(
      `${BASE_URL}/douyin/like/action/?video_id=${videoId}`,
      null,
      authHeaders,
    );
    check(like, { "like action ok": (r) => r.status === 200 });
  }

  sleep(1);
}
