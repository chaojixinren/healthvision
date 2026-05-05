FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /server ./cmd/server/

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

ENV TZ=Asia/Shanghai
ENV APP_ENV=production
ENV PORT=8080

COPY --from=builder /server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
