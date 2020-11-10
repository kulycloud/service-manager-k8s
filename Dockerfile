FROM golang:1.15.3-alpine AS builder

ADD service-manager-k8s/go.mod service-manager-k8s/go.sum /build/service-manager-k8s/
ADD protocol/go.mod protocol/go.sum /build/protocol/
ADD common/go.mod common/go.sum /build/common/

ENV CGO_ENABLED=0

WORKDIR /build/service-manager-k8s
RUN go mod download

COPY service-manager-k8s/ /build/service-manager-k8s/
COPY protocol/ /build/protocol
COPY common/ /build/common
RUN go build -o /build/kuly .

FROM scratch

COPY --from=builder /build/kuly /

CMD ["/kuly"]
