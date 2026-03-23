FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/dashyreborn .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/dashyreborn /usr/local/bin/dashyreborn
COPY conf.sample.yml /app/conf.sample.yml
COPY readme.md /app/README.md

RUN mkdir -p /app/.cache/favicons

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/dashyreborn"]
CMD ["-addr", "0.0.0.0:8080", "-config", "/app/conf.sample.yml", "-assets-mode", "auto"]

#docker run --privileged --rm tonistiigi/binfmt --install all
#docker buildx build --platform linux/amd64,linux/arm64 --push -t aejii/dashyreborn:1.0.0 .
