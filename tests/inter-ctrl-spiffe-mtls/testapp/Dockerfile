# syntax=docker/dockerfile:1
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN CGO_ENABLED=0 go build -o /out/testapp .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/testapp /testapp
ENTRYPOINT ["/testapp"]
