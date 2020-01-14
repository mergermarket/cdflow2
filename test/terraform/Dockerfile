FROM golang:latest
WORKDIR /app
ADD . .
RUN go build -o /bin/terraform .
ENTRYPOINT ["/bin/terraform"]