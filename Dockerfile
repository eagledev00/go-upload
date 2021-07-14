FROM golang:alpine as builder

WORKDIR /app

ADD . /app

RUN apk add --no-cache git
RUN go get -u github.com/gorilla/mux/... github.com/gorilla/handlers/... github.com/gobuffalo/packr/...
RUN CGO_ENABLED=0 GOOS=linux packr build -a -installsuffix cgo -o go-upload .

FROM scratch

COPY --from=builder /app/go-upload .

EXPOSE 9000

ENV LISTEN_ADDRESS 0.0.0.0:9000
ENV PUBLIC_ROOT http://127.0.0.1:9000/
ENV MAX_UPLOAD_SIZE_IN_MB 200
ENV UPLOAD_KEY DEFAULT_KEY
ENV STORAGE_PATH /data
ENV ENABLE_WEBFORM true
ENV FILENAME_LENGTH 6

VOLUME /data
WORKDIR /data

CMD [ "/go-upload" ]
