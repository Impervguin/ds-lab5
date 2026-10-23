FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/gateway ./cmd

FROM alpine:3.20
WORKDIR /app

COPY --from=build /out/gateway ./gateway

EXPOSE 8080
ENTRYPOINT ["./gateway"]
