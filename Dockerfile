FROM golang:alpine as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o main main.go

# Clean image on amazon linux 2023 lambda
FROM public.ecr.aws/lambda/provided:al2023

ARG MAPS_KEY
ARG BASE_PATH

ENV MAPS_KEY=${MAPS_KEY}
ENV BASE_PATH=${BASE_PATH}
ENV ENVIRONMENT="production"

COPY --from=builder /app/main ./main
ENTRYPOINT ["./main"]