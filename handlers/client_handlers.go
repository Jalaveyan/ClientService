package handlers

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
)

type Client struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Email   string `json:"email"`
	Comment string `json:"comment,omitempty"`
}

func validatePhone(phone string) bool {
	re := regexp.MustCompile(`^\+?\d{10,15}$`)
	return re.MatchString(phone)
}

func validateEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// CreateClient добавляет нового клиента
func CreateClient(db *pgxpool.Pool, logger *zap.SugaredLogger) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var client Client
		if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
			logger.Errorf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		logger.Infof("Received data for new client: %+v", client)

		if !validatePhone(client.Phone) || !validateEmail(client.Email) {
			logger.Warnf("Validation failed for phone: %s or email: %s", client.Phone, client.Email)
			http.Error(w, "Invalid phone or email format", http.StatusBadRequest)
			return
		}

		// Проверка длины комментария
		if len(client.Comment) > 255 {
			logger.Warnf("Comment is too long: %d characters", len(client.Comment))
			http.Error(w, "Comment is too long", http.StatusBadRequest)
			return
		}

		client.ID = uuid.New().String()
		logger.Infof("Generated UUID: %s", client.ID)

		query := `INSERT INTO clients (id, name, phone, email, comment) VALUES ($1, $2, $3, $4, $5)`
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := db.Exec(ctx, query, client.ID, client.Name, client.Phone, client.Email, client.Comment)
		if err != nil {
			logger.Errorf("Error inserting client into database: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		logger.Info("Client successfully created")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(client)
	}
}

// GetClientByID получает клиента по ID
func GetClientByID(db *pgxpool.Pool, logger *zap.SugaredLogger) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id := ps.ByName("id")
		var client Client

		query := `SELECT id, name, phone, email, comment FROM clients WHERE id = $1`
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := db.QueryRow(ctx, query, id).Scan(&client.ID, &client.Name, &client.Phone, &client.Email, &client.Comment)
		if err != nil {
			if err == pgx.ErrNoRows {
				http.Error(w, "Client not found", http.StatusNotFound)
				return
			}
			logger.Errorf("Error querying client: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(client)
	}
}

// GetClients получает список клиентов с пагинацией
func GetClients(db *pgxpool.Pool, logger *zap.SugaredLogger) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		limit := 10
		offset := 0

		if l := r.URL.Query().Get("limit"); l != "" {
			val, err := strconv.Atoi(l)
			if err != nil || val < 0 {
				logger.Warnf("Invalid limit value: %s", l)
				http.Error(w, "Invalid limit value", http.StatusBadRequest)
				return
			}
			limit = val
		}

		if o := r.URL.Query().Get("offset"); o != "" {
			val, err := strconv.Atoi(o)
			if err != nil || val < 0 {
				logger.Warnf("Invalid offset value: %s", o)
				http.Error(w, "Invalid offset value", http.StatusBadRequest)
				return
			}
			offset = val
		}

		logger.Infof("Fetching clients with limit: %d and offset: %d", limit, offset)

		query := `SELECT id, name, phone, email, comment FROM clients LIMIT $1 OFFSET $2`
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := db.Query(ctx, query, limit, offset)
		if err != nil {
			logger.Errorf("Error fetching clients: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		clients := []Client{}
		for rows.Next() {
			var client Client
			if err := rows.Scan(&client.ID, &client.Name, &client.Phone, &client.Email, &client.Comment); err != nil {
				logger.Errorf("Error scanning client row: %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			clients = append(clients, client)
		}

		logger.Infof("Fetched %d clients", len(clients))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(clients)
	}
}

// UpdateClient обновляет данные клиента
func UpdateClient(db *pgxpool.Pool, logger *zap.SugaredLogger) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id := ps.ByName("id")
		logger.Infof("Updating client with ID: %s", id)

		var client Client
		if err := json.NewDecoder(r.Body).Decode(&client); err != nil {
			logger.Errorf("Error decoding JSON: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Валидация входных данных
		if !validateEmail(client.Email) {
			http.Error(w, "Invalid email format", http.StatusBadRequest)
			return
		}

		if !validatePhone(client.Phone) {
			http.Error(w, "Invalid phone format", http.StatusBadRequest)
			return
		}

		if len(client.Comment) > 255 {
			http.Error(w, "Comment is too long", http.StatusBadRequest)
			return
		}

		query := `UPDATE clients SET name = $1, phone = $2, email = $3, comment = $4 WHERE id = $5`
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmdTag, err := db.Exec(ctx, query, client.Name, client.Phone, client.Email, client.Comment, id)
		if err != nil {
			logger.Errorf("Error updating client: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if cmdTag.RowsAffected() == 0 {
			logger.Warnf("Client with ID %s not found", id)
			http.Error(w, "Client not found", http.StatusNotFound)
			return
		}

		logger.Infof("Successfully updated client: %+v", client)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(client)
	}
}

// DeleteClient удаляет клиента
func DeleteClient(db *pgxpool.Pool, logger *zap.SugaredLogger) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		id := ps.ByName("id")
		logger.Infof("Deleting client with ID: %s", id)

		query := `DELETE FROM clients WHERE id = $1`
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmdTag, err := db.Exec(ctx, query, id)
		if err != nil {
			logger.Errorf("Error deleting client: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if cmdTag.RowsAffected() == 0 {
			logger.Warnf("Client with ID %s not found", id)
			http.Error(w, "Client not found", http.StatusNotFound)
			return
		}

		logger.Infof("Successfully deleted client with ID: %s", id)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Client deleted"))
	}
}
