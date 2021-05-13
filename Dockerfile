FROM golang AS builder

RUN mkdir -p /go/src/github.com/lunemec/eve-fuelbot
WORKDIR /go/src/github.com/lunemec/eve-fuelbot
COPY . .

RUN go get github.com/ahmetb/govvv
RUN CGO_ENABLED=0 GOOS=linux govvv build -pkg github.com/lunemec/eve-fuelbot/pkg/version -o fuelbot

FROM scratch

COPY --from=builder /go/src/github.com/lunemec/eve-fuelbot/fuelbot .
ENTRYPOINT [ "/fuelbot" ]
