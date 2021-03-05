FROM golang:1.15.3-alpine AS builder

ADD go.mod go.sum /build/

ENV CGO_ENABLED=0

WORKDIR /build
RUN go mod download

COPY ./ /build/
RUN go build -o /build/kuly .

FROM scratch

COPY --from=builder /build/kuly /

CMD ["/kuly"]
