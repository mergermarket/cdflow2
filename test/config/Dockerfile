FROM golang:alpine AS build
WORKDIR /
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .
RUN mkdir /empty

FROM scratch
COPY --from=build /empty /tmp
ENV TMPDIR /tmp
COPY --from=build /app /app
ENTRYPOINT ["/app"]