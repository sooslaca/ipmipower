FROM golang:alpine AS builder

RUN apk update && apk add --no-cache git

WORKDIR $GOPATH/src/sooslaca/ipmipower/
COPY . ./

RUN go mod download

RUN CGO_ENABLED=0 go build -o /go/bin/ipmipower -v -trimpath -ldflags "-s -w" .



FROM scratch

COPY --from=builder /go/bin/ipmipower /ipmipower

ENTRYPOINT ["/ipmipower"]
