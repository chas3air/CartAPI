ARG GO_VERSION=latest
FROM golang:${GO_VERSION} AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETARCH
RUN CGO_ENABLED=0 GOARCH=${TARGETARCH} go build -o /cli ./cmd/app

FROM alpine:latest AS final

WORKDIR /

RUN apk --no-cache add ca-certificates tzdata

COPY --from=build /cli /cli

COPY --from=build /src/migrations ./migrations
COPY config.yaml config.yaml

EXPOSE 8080

# ENTRYPOINT ["tail", "-f", "/dev/null"]
CMD ["./cli"]