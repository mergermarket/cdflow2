FROM golang:alpine AS build
WORKDIR /
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /bin/terraform .

# from alpine rather than scratch as we require /bin/sleep (this isn't a standalone binary that can be copied)
FROM alpine
COPY --from=build /bin/terraform /bin/terraform
ENTRYPOINT ["/bin/terraform"]