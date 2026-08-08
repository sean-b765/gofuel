FROM golang:alpine as builder

ARG FUNCTION=api/prod
WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN go build -o main ./cmd/${FUNCTION}

# Clean image on amazon linux 2023 lambda
FROM public.ecr.aws/lambda/provided:al2023

ARG BASE_PATH

ENV BASE_PATH=${BASE_PATH}
ENV ENVIRONMENT="production"

COPY --from=builder /app/main ./main
ENTRYPOINT ["./main"]