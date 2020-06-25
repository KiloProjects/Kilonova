FROM golang:1.14

COPY ./ /app

WORKDIR /app

ENTRYPOINT [ "/app/Kilonova"]