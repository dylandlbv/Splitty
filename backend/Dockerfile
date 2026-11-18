# build stage
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /bin/splitty ./cmd/splitty

# runtime stage
FROM alpine:3.20
RUN adduser -D -u 10001 splitty
USER splitty
COPY --from=build /bin/splitty /bin/splitty
EXPOSE 8080
ENTRYPOINT ["/bin/splitty"]
