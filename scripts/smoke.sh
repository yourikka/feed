#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:3000}"
API_URL="${API_URL:-http://localhost:8080}"

echo "[smoke] checking frontend: ${BASE_URL}"
curl -fsS "${BASE_URL}/" >/dev/null

echo "[smoke] checking backend feed latest"
curl -fsS "${API_URL}/douyin/feed/?sort=latest&limit=2" >/dev/null

echo "[smoke] checking backend feed hot"
curl -fsS "${API_URL}/douyin/feed/?sort=hot&limit=2" >/dev/null

echo "[smoke] all checks passed"
