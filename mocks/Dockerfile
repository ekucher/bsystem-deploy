# Shared image for every BSYSTEM mock upstream.
#
# Select the binary with --build-arg MOCK=mock-espocrm (or mock-identity,
# mock-redmine, mock-outline).
FROM golang:1.26-alpine AS build

ARG MOCK
WORKDIR /src

COPY go.mod ./
COPY internal ./internal
COPY cmd ./cmd

RUN test -n "$MOCK" || (echo "MOCK build argument is required" >&2; exit 1)
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mock ./cmd/${MOCK}

# scratch keeps the mocks free of a shell and a package manager. The binary
# healthchecks itself with `mock -healthcheck`.
FROM scratch

COPY --from=build /out/mock /mock

# 65532 is the conventional nonroot uid; the mocks need no filesystem access.
USER 65532:65532

ENTRYPOINT ["/mock"]
