FROM golang:latest
WORKDIR /
ADD . .
RUN go build -o /app .
ENTRYPOINT ["/app"]