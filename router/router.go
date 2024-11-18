package router

import (
	"client_service/handlers"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
	"net/http"
)

// InitRouter инициализирует маршруты и принимает подключение к базе данных и логгер
func InitRouter(db *pgxpool.Pool, logger *zap.SugaredLogger) *httprouter.Router {
	router := httprouter.New()

	//crud
	router.POST("/clients", handlers.CreateClient(db, logger))
	router.GET("/clients/:id", handlers.GetClientByID(db, logger))
	router.GET("/clients", handlers.GetClients(db, logger))
	router.PUT("/clients/:id", handlers.UpdateClient(db, logger))
	router.DELETE("/clients/:id", handlers.DeleteClient(db, logger))

	router.GET("/", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Service is healthy"))
	})

	return router
}
