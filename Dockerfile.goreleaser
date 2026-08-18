ARG GO_VERSION=1.26.2

FROM golang:${GO_VERSION}-alpine

ARG TARGETPLATFORM
COPY $TARGETPLATFORM/go-crap /usr/local/bin/go-crap

WORKDIR /code

ENTRYPOINT ["go-crap"]
CMD ["scan", "/code"]
