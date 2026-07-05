FROM golang:alpine AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/tracker .

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /

COPY --from=build /out/tracker /tracker

VOLUME ["/data"]
ENV STATE_FILE=/data/state.json

ENTRYPOINT ["/tracker"]
