#!/usr/bin/env bash
# Push the app over the HPA CPU target for a couple of minutes.
# Usage: scripts/load.sh [url] [seconds]
set -u

URL="${1:-http://localhost:8080}"
SECONDS_RUN="${2:-120}"
WORKERS=12

echo "HPA before load:"
kubectl -n skypod get hpa

echo
echo "Generating load against ${URL}/work?ms=200 for ${SECONDS_RUN}s with ${WORKERS} workers..."

load_one() {
  local end=$((SECONDS + SECONDS_RUN))
  while [ $SECONDS -lt $end ]; do
    curl -s -o /dev/null "${URL}/work?ms=200" || true
  done
}

# The script port-forwards the service itself, so it works from any machine.
kubectl -n skypod port-forward svc/skypod 8080:80 >/dev/null 2>&1 &
PF_PID=$!
trap 'kill $PF_PID 2>/dev/null' EXIT
sleep 3

for _ in $(seq 1 "$WORKERS"); do
  load_one &
done
wait

echo
echo "HPA after load:"
kubectl -n skypod get hpa
