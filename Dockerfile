FROM golang:1.14.10-alpine3.12
WORKDIR /app
COPY . .
RUN go build -o containermon .

FROM alpine:3.12
WORKDIR /app
COPY --from=0 /app/containermon .
CMD ["./containermon"]