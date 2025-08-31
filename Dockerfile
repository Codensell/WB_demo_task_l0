FROM golang:1.25.0

WORKDIR /usr/app/

COPY go.mod go.sum ./
RUN go mod download

COPY . .

CMD go run ./cmd/app