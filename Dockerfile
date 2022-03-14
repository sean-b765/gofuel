FROM golang:alpine as builder

WORKDIR /go/app

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build


FROM alpine

WORKDIR /app

COPY --from=builder /go/app /app
ENV PORT 8000
CMD ["./fuel"]