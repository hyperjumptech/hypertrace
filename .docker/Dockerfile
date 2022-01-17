FROM golang:1.17
ENV GOPATH /go
ENV GO111MODULE on
ENV GOOS linux
ENV GOARCH amd64

# Prepare all the dirs.
RUN mkdir -p $GOPATH/src/github.com/hyperjumptech/hypertrace
# Copy the build content.
COPY . $GOPATH/src/github.com/hyperjumptech/hypertrace
# Checkout the go-resource to auto generate statics into go codes.
WORKDIR $GOPATH/src/github.com/hyperjumptech/hypertrace
# Compile the proje ct
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o HTrace.app cmd/main.go

# Now use the deployment image.
FROM alpine:latest
ENV GOPATH /go
RUN apk --no-cache add ca-certificates
# Copy the built binary to the new image.
WORKDIR /root/
COPY --from=0 $GOPATH/src/github.com/hyperjumptech/hypertrace/HTrace.app .
# Expose port.
EXPOSE 8080
# Execute
CMD ["./HTrace.app"]