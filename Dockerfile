ARG APP_NAME="tfs"
ARG PORT=3000

# Fetch
FROM golang:latest AS fetch-stage

COPY go.mod go.sum /app
WORKDIR /app
RUN go mod download

FROM node:22-alpine AS tailwind-prebuild
COPY . /app/
RUN corepack enable
WORKDIR /app/
RUN yarn install
RUN npx tailwindcss -i ./static/css/input.css -o ./static/css/style.min.css --minify

# Generate
FROM ghcr.io/a-h/templ:latest AS templ-prebuild
COPY --chown=65532:65532 . /app
WORKDIR /app
RUN ["templ", "generate"] 

FROM golang:1.23.5-alpine AS build
ARG APP_NAME

COPY . /app
WORKDIR /app

RUN go build -ldflags "-X main.Environment=production" -o ./bin/$APP_NAME ./main.go
RUN rm ./static/css/input.css

FROM alpine:latest AS run
ARG APP_NAME
ARG PORT
ENV MODE="PRODUCTION"

COPY --from=tailwind-prebuild /app/static/css/style.min.css /app/static/css/style.css
COPY --from=build /app/bin/$APP_NAME /app/tfs
COPY .env /app/.env

WORKDIR /

EXPOSE $PORT
ENTRYPOINT ["/app/tfs"]
