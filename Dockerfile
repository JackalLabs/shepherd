# syntax=docker/dockerfile:1

FROM golang:1.21

# Set destination for COPY
WORKDIR /app

# Download Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code. Note the slash at the end, as explained in
# https://docs.docker.com/engine/reference/builder/#copy
COPY *.go ./
COPY *.json ./
# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /shepherd

# To bind to a TCP port, runtime parameters must be supplied to the docker command.
# But we can (optionally) document in the Dockerfile what ports
# the application is going to listen on by default.
# https://docs.docker.com/engine/reference/builder/#expose
EXPOSE 5656

ENV RPC "https://rpc.jackalprotocol.com:443"
ENV PORT "5656"

# Run
CMD [ "/shepherd" ]