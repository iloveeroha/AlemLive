FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/alemlive-backend ./cmd/server

FROM alpine:3.23

RUN adduser -D -H -u 10001 appuser
WORKDIR /app

COPY --from=build /out/alemlive-backend /app/alemlive-backend

USER appuser
EXPOSE 8080

CMD ["/app/alemlive-backend"]
