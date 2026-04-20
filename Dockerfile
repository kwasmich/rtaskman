FROM golang:1.26-alpine AS build

WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o rtaskman ./cmd/rtaskman





FROM scratch

COPY --from=build /src/rtaskman /rtaskman
# COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV PORT=8087

EXPOSE 8087

ENTRYPOINT ["/rtaskman"]
