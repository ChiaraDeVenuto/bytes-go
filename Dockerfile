# Multistage build for portmortem/bytes-go
# Produces a static, dependency-free binary (~7 MB).

FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
COPY bytefmt.go ./
COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/bytefmt ./cmd/bytefmt

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/bytefmt /usr/local/bin/bytefmt
ENTRYPOINT ["/usr/local/bin/bytefmt"]
