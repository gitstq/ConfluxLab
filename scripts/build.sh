#!/usr/bin/env bash
# 一键跨平台构建：纯 Go 标准库，CGO_ENABLED=0，产出无运行时依赖的静态二进制。
set -euo pipefail

VERSION="${VERSION:-v1.0.0}"
OUT="${OUT:-dist}"
mkdir -p "$OUT"

TARGETS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

echo "ConfluxLab 跨平台构建 · ${VERSION}"
for target in "${TARGETS[@]}"; do
  GOOS="${target%/*}"
  GOARCH="${target#*/}"
  ext=""
  if [ "$GOOS" = "windows" ]; then ext=".exe"; fi
  bin="confluxlab-${VERSION}-${GOOS}-${GOARCH}${ext}"
  printf '  → %-46s' "$bin"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o "${OUT}/${bin}" ./cmd/confluxlab
  echo "OK"
done

(cd "$OUT" && sha256sum confluxlab-* > SHA256SUMS.txt)
echo "校验和已写入 ${OUT}/SHA256SUMS.txt"
echo "本机二进制自检："
"${OUT}/confluxlab-${VERSION}-linux-amd64" version
