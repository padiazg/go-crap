ARG GO_VERSION=1.26.2

FROM golang:${GO_VERSION}-alpine AS builder

ARG VERSION=0.0.0-dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .

ARG TARGETOS TARGETARCH
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -ldflags "-s -w \
      -X github.com/padiazg/go-crap/pkg/version.version=${VERSION} \
      -X github.com/padiazg/go-crap/pkg/version.commit=${COMMIT} \
      -X github.com/padiazg/go-crap/pkg/version.buildDate=${BUILD_DATE}" \
      -o /go-crap ./main.go

FROM golang:${GO_VERSION}-alpine

COPY --from=builder /go-crap /usr/local/bin/go-crap

WORKDIR /code

ENTRYPOINT ["go-crap"]
CMD ["scan", "/code"]
