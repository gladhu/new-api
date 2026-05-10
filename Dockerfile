# Build parallelism: reserve RESERVE_CPU_CORES for the host (default 1). Example:
#   docker build --build-arg RESERVE_CPU_CORES=2 .
# Default/classic frontend stages may run in parallel; each uses roughly
# floor((nproc - RESERVE) / 2) via GOMAXPROCS so the pair is less likely to
# saturate all CPUs. The final Go compile stage uses (nproc - RESERVE).
FROM oven/bun:1@sha256:0733e50325078969732ebe3b15ce4c4be5082f18c4ac1a0f0ca4839c2e4e42a7 AS builder
ARG RESERVE_CPU_CORES=1

WORKDIR /build
COPY web/default/package.json .
COPY web/default/bun.lock .
RUN bun install
COPY ./web/default .
COPY ./VERSION .
RUN RESERVE="${RESERVE_CPU_CORES:-1}"; \
    TOTAL=$(nproc); \
    if [ "$TOTAL" -gt "$RESERVE" ]; then AVAIL=$((TOTAL - RESERVE)); else AVAIL=1; fi; \
    USE=$((AVAIL / 2)); \
    if [ "$USE" -lt 1 ]; then USE=1; fi; \
    export GOMAXPROCS="$USE"; \
    DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

FROM oven/bun:1@sha256:0733e50325078969732ebe3b15ce4c4be5082f18c4ac1a0f0ca4839c2e4e42a7 AS builder-classic
ARG RESERVE_CPU_CORES=1

WORKDIR /build
COPY web/classic/package.json .
COPY web/classic/bun.lock .
RUN bun install
COPY ./web/classic .
COPY ./VERSION .
RUN RESERVE="${RESERVE_CPU_CORES:-1}"; \
    TOTAL=$(nproc); \
    if [ "$TOTAL" -gt "$RESERVE" ]; then AVAIL=$((TOTAL - RESERVE)); else AVAIL=1; fi; \
    USE=$((AVAIL / 2)); \
    if [ "$USE" -lt 1 ]; then USE=1; fi; \
    export GOMAXPROCS="$USE"; \
    VITE_REACT_APP_VERSION=$(cat VERSION) bun run build

FROM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder2
ENV GO111MODULE=on CGO_ENABLED=0

ARG RESERVE_CPU_CORES=1
ARG TARGETOS
ARG TARGETARCH
ENV GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64}
ENV GOEXPERIMENT=greenteagc

WORKDIR /build

ADD go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=builder /build/dist ./web/default/dist
COPY --from=builder-classic /build/dist ./web/classic/dist
RUN RESERVE="${RESERVE_CPU_CORES:-1}"; \
    TOTAL=$(nproc); \
    if [ "$TOTAL" -gt "$RESERVE" ]; then USE=$((TOTAL - RESERVE)); else USE=1; fi; \
    export GOMAXPROCS="$USE"; \
    go build -ldflags "-s -w -X 'github.com/QuantumNous/new-api/common.Version=$(cat VERSION)'" -o new-api

FROM debian:bookworm-slim@sha256:f06537653ac770703bc45b4b113475bd402f451e85223f0f2837acbf89ab020a

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libasan8 wget \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

COPY --from=builder2 /build/new-api /
EXPOSE 3000
WORKDIR /data
ENTRYPOINT ["/new-api"]
