FROM golang:1.10.2 as builder
# install dep
RUN go get -u github.com/golang/dep/cmd/dep
# setup the working directory
WORKDIR /go/src/app
COPY . .
# install dependencies
RUN dep ensure
# build the source
RUN cd cmd/spd && CGO_ENABLED=0 GOOS=linux go build

# use a minimal image
FROM ubuntu:16.04
# set working directory
WORKDIR /root
# copy the binary from builder
COPY --from=builder /go/src/app/cmd/spd/config.yml /go/src/app/cmd/spd/prices_test.yml /go/src/app/cmd/spd/spd ./
# run the binary
CMD ["./spd"]
