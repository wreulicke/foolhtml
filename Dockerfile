FROM golang:1.26 AS builder

WORKDIR /build

RUN --mount=type=bind,target=. CGO_ENABLED=0 go build -o /app/foolhtml ./cmd/foolhtml

FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/foolhtml /app/foolhtml

ENTRYPOINT [ "/app/foolhtml" ]