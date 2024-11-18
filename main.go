package main

import (
	"client_service/router"
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
)

// Глобальная переменная для доступа к базе данных
var DB *pgxpool.Pool

func main() {
	// Инициализация Zap
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Unable to initialize logger: %v", err)
	}
	defer logger.Sync()
	sugar := logger.Sugar()

	// Получение строки подключения из переменной окружения
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		sugar.Fatal("DATABASE_URL environment variable is not set")
	}

	sugar.Infof("Connecting to database using DSN: %s", dsn)

	ctx := context.Background()

	DB, err = pgxpool.New(ctx, dsn)
	if err != nil {
		sugar.Fatalf("Unable to connect to database: %v", err)
	}
	defer DB.Close()

	sugar.Info("Connected to database.")

	// Инициализация маршрутов с логгером
	r := router.InitRouter(DB, sugar)

	// Запуск HTTP-сервера
	sugar.Info("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
