FROM golang:alpine as builder

WORKDIR /app

ARG MAPS_KEY
ARG BASE_PATH

ENV MAPS_KEY=${MAPS_KEY}
ENV BASE_PATH=${BASE_PATH}

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o main main.go

# Clean image on amazon linux 2023 lambda
FROM public.ecr.aws/lambda/provided:al2023

COPY --from=builder /app/main ./main
ENTRYPOINT ["./main"]