FROM golang:1.10.2 as builder
# install dep
RUN go get -u github.com/golang/dep/cmd/dep
# setup the working directory
WORKDIR /go/src/app
COPY . .
# install dependencies
RUN dep ensure
# build the source
RUN CGO_ENABLED=0 GOOS=linux go build

# use a minimal image
FROM ubuntu:16.04
# set working directory
WORKDIR /root
# copy the binary from builder
COPY --from=builder /go/src/app/config.yml /go/src/app/SPDT.exe ./
# Document that the service listens on port 8080.
EXPOSE 8082
# Run the  command by default when the container starts.
CMD ["./SPDT"]
