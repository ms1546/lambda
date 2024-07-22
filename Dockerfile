FROM amazonlinux:2023

RUN yum update -y && \
    dnf -y install mariadb105 &&\
    yum install -y golang git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o main main.go

CMD ["./main"]
