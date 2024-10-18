FROM golang
WORKDIR /app
RUN mkdir cache
COPY go.mod go.sum .
RUN go mod download
COPY *.go tpl.html .
RUN go build
CMD ./cars
