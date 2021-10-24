FROM golang:alpine
ENV CGO_ENABLED 0 
RUN apk update && apk add --no-cache git curl
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN curl -sSfL https://raw.githubusercontent.com/cosmtrek/air/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
EXPOSE 8080
EXPOSE 5432
ENTRYPOINT ["air"]