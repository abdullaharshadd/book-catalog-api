FROM golang:1.23-alpine

WORKDIR /app

COPY . .

EXPOSE 8080

CMD ["sh", "-c", ""]
