FROM golang:1.17-alpine as builder
WORKDIR /src
COPY ./ .
RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/main.go


FROM gcr.io/distroless/static
WORKDIR /opt/bin
COPY --from=builder /src/server .
ENTRYPOINT ["./server"]
