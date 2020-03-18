FROM golang:latest


COPY ./ /app

WORKDIR /app

RUN go get github.com/githubnemo/CompileDaemon

ENTRYPOINT CompileDaemon --build="go build" --command=./kilonova