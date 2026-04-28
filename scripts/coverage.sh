#!/usr/bin/env sh
set -eu

threshold="${COVERAGE_THRESHOLD:-95.0}"
raw_profile="${COVERAGE_RAW_PROFILE:-coverage.out}"
profile="${COVERAGE_PROFILE:-coverage.filtered.out}"

go test ./... -coverprofile="$raw_profile"

# The command entrypoint starts real infrastructure and is validated by build
# checks. Coverage is enforced for application packages, including generated
# sqlc query wrappers.
grep -v '/cmd/conevent/' "$raw_profile" > "$profile"

coverage="$(go tool cover -func="$profile" | awk '/^total:/ {gsub("%", "", $3); print $3}')"
awk -v coverage="$coverage" -v threshold="$threshold" 'BEGIN {
  if (coverage + 0 < threshold + 0) {
    printf("coverage %.1f%% is below required %.1f%%\n", coverage, threshold)
    exit 1
  }
  printf("coverage %.1f%% meets required %.1f%%\n", coverage, threshold)
}'
