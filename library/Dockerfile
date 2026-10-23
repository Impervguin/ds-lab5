FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/library ./cmd

FROM alpine:3.20
WORKDIR /app

# The migrations are embedded into the binary (internal/migrations), so the
# image needs nothing but the binary itself.
COPY --from=build /out/library ./library

EXPOSE 8060
ENTRYPOINT ["./library"]
