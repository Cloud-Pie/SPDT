FROM golang:latest as builder
# install dep
RUN go get github.com/golang/dep/cmd/dep

# setup the working directory
WORKDIR /go/src/spdt
COPY . .
# install dependencies
RUN dep ensure -v
# build the source
RUN CGO_ENABLED=0 GOOS=linux go build

# use a minimal image
FROM ubuntu:16.04
# set working directory
WORKDIR /root

# copy the binary and default config files from builder
COPY --from=builder /go/src/spdt/config.yml /go/src/spdt/spdt ./
COPY --from=builder /go/src/spdt/ui ./ui/

# Document that the service listens on port 8080.
EXPOSE 8080
# Run the  command by default when the container starts.
CMD ["./spdt"]
