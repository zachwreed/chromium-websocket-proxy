FROM alpine:3.18.5

WORKDIR /app

RUN apk update
RUN apk --no-cache add chromium~=119.0

# make sure before running this that GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/chromium-websocket-proxy . has been ran.
# Or you can install Golang here and build in the container. Whatever you prefer :)

COPY build/chromium-websocket-proxy chromium-websocket-proxy

# Uncommment to use a custom profile for your build
#COPY profiles/profile.zip profiles/profile.zip

ENTRYPOINT ["./chromium-websocket-proxy"]