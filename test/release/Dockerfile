FROM golang:1.19-alpine AS build
WORKDIR /
ADD . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

FROM scratch
COPY --from=build /app /app
ENTRYPOINT ["/app"]
