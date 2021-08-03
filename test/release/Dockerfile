FROM golang:1.13.15-alpine3.12 AS build
WORKDIR /
ADD . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM scratch
COPY --from=build /app /app
ENTRYPOINT ["/app"]