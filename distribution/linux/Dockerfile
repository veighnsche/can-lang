# T19 Linux build image. The base is pinned by digest; the Go toolchain is
# pinned by URL and SHA-256 (see provenance.json). Debian apt packages are
# installed without version pins; build.sh records the resolved versions
# into the run report. Build from a clean checkout:
#
#   docker build --platform linux/amd64 -f distribution/linux/Dockerfile -t can-linux/t19 .
FROM docker.io/library/debian:13@sha256:d5ce19d4736f0ebbacd686d1040271a5aeb0cc920f5990c1bfae1717627f0674

ENV DEBIAN_FRONTEND=noninteractive \
    GOTOOLCHAIN=local \
    GOPROXY=https://proxy.golang.org,direct \
    PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl gcc libc6-dev python3 binutils \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL -o /tmp/go.tgz https://go.dev/dl/go1.27.1.linux-amd64.tar.gz \
    && echo "63d339f0da5ab53635a56f2490a7984dfe12dfcff22ad749f63edaf590168445  /tmp/go.tgz" | sha256sum -c - \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm /tmp/go.tgz \
    && go version

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
