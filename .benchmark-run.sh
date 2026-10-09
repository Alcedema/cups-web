#!/bin/sh
set -eu
mkdir -p .bench
measure() {
  label=$1; mode=$2; rep=$3; shift 3
  start=$(awk '{print $1}' /proc/uptime)
  "$@"
  end=$(awk '{print $1}' /proc/uptime)
  elapsed=$(awk -v a="$start" -v b="$end" 'BEGIN {printf "%.2f", b-a}')
  printf 'BENCH_RESULT {"workload":"%s","mode":"%s","rep":%s,"seconds":%s,"runner":"%s","job":%s}\n' "$label" "$mode" "$rep" "$elapsed" "$CI_RUNNER_DESCRIPTION" "$CI_JOB_ID" | tee -a .bench/results.jsonl
}
case "$BENCH_WORKLOAD" in
frontend)
  for rep in 1 2 3; do
    export npm_config_cache="$CI_PROJECT_DIR/.bench/npm-$rep"
    for mode in cold warm; do
      measure frontend "$mode" "$rep" sh -ec 'cd frontend; npx --yes npm@11.6.2 ci --prefer-offline; npm run check:translations; npm test; npm run build; cmp ../LICENSE dist/LICENSE.txt'
    done
    rm -rf "$CI_PROJECT_DIR/.bench/npm-$rep"
  done
  ;;
backend)
  export CGO_ENABLED=0 GOMAXPROCS=4
  mkdir -p bin
  for rep in 1 2 3; do
    export GOMODCACHE="$CI_PROJECT_DIR/.bench/go-mod-$rep" GOCACHE="$CI_PROJECT_DIR/.bench/go-build-$rep"
    for mode in cold warm; do
      measure backend "$mode" "$rep" sh -ec 'go test -count=1 ./...; go vet ./...; go build -trimpath -ldflags "-s -w" -o bin/benchmark-server ./cmd/server'
    done
    chmod -R u+w "$GOMODCACHE"
    rm -rf "$GOMODCACHE" "$GOCACHE"
  done
  ;;
ansible)
  export ANSIBLE_CONFIG="$CI_PROJECT_DIR/ansible.cfg" PIP_DISABLE_PIP_VERSION_CHECK=1
  for rep in 1 2 3; do
    export PIP_CACHE_DIR="$CI_PROJECT_DIR/.bench/pip-$rep"
    export ANSIBLE_COLLECTIONS_PATH="$CI_PROJECT_DIR/.bench/collections-$rep"
    for mode in cold warm; do
      measure ansible "$mode" "$rep" sh -ec 'python -m venv .bench/venv; . .bench/venv/bin/activate; pip install -r requirements-dev.txt; ansible-galaxy collection install -r collections/requirements.yml; make check'
      rm -rf "$CI_PROJECT_DIR/.bench/venv"
    done
    rm -rf "$PIP_CACHE_DIR" "$ANSIBLE_COLLECTIONS_PATH"
  done
  ;;
image-*)
  image=${BENCH_WORKLOAD#image-}
  for rep in 1 2 3; do
    root="/home/user/.local/share/buildkit/benchmark-$CI_JOB_ID-$rep"
    export BUILDKITD_FLAGS="--oci-worker-no-process-sandbox --oci-worker-snapshotter=overlayfs --root=$root"
    for mode in cold warm; do
      measure "$BENCH_WORKLOAD" "$mode" "$rep" buildctl-daemonless.sh build --frontend dockerfile.v0 --local "context=images/$image" --local "dockerfile=images/$image" --opt platform=linux/amd64 --output "type=oci,dest=.bench/image.tar"
      rm -f .bench/image.tar
    done
    rm -rf "$root"
  done
  ;;
*) exit 2;;
esac
