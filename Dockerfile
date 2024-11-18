# Stage 1: Build
FROM golang:1.22.4 AS builder

WORKDIR /app

# Установить утилиту file
RUN apt-get update && apt-get install -y file

# Копируем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходный код
COPY . .

# Компилируем для Linux
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

# Проверяем бинарный файл
RUN file main

# Stage 2: Runtime
FROM alpine:latest

# Устанавливаем сертификаты
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /app/main .

# Устанавливаем права на выполнение
RUN chmod +x ./main

EXPOSE 8080
CMD ["./main"]
