FROM golang:1.22
WORKDIR /x
COPY go.mod go.sum .
RUN go mod download
COPY main.go .
RUN go build -o /diag .
ENTRYPOINT ["/diag"]
