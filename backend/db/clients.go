package db

import (
	"log"
	"net/http"
	"strings"

	"github.com/Talonmortem/SHM/internal/models"
	"github.com/gin-gonic/gin"
)

func CreateTablesClients() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS clients (
			id BIGSERIAL PRIMARY KEY,
			city TEXT,
			full_name TEXT NOT NULL,
			phone TEXT,
			passport_number TEXT,
			tk TEXT,
			comment TEXT
		);

		CREATE INDEX IF NOT EXISTS idx_clients_city ON clients(city);
		CREATE INDEX IF NOT EXISTS idx_clients_full_name ON clients(full_name);
	`)
	if err != nil {
		log.Fatal("Failed to create clients table:", err)
	}
	log.Println("Clients table is ready")
}

func CreateClient(c *gin.Context) {
	var client models.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(client.FullName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО обязательно"})
		return
	}

	err := DB.QueryRow(`
		INSERT INTO clients (city, full_name, phone, passport_number, tk, comment)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, client.City, client.FullName, client.Phone, client.PassportNumber, client.TK, client.Comment).Scan(&client.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, client)
}

func GetClients(c *gin.Context) {
	rows, err := DB.Query(`
		SELECT id, city, full_name, phone, passport_number, tk, comment
		FROM clients
		ORDER BY id DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	clients := make([]models.Client, 0)
	for rows.Next() {
		var client models.Client
		if err := rows.Scan(&client.ID, &client.City, &client.FullName, &client.Phone, &client.PassportNumber, &client.TK, &client.Comment); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		clients = append(clients, client)
	}

	c.JSON(http.StatusOK, clients)
}

func UpdateClient(c *gin.Context) {
	id := c.Param("id")
	var client models.Client
	if err := c.ShouldBindJSON(&client); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(client.FullName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО обязательно"})
		return
	}

	_, err := DB.Exec(`
		UPDATE clients
		SET city = $1, full_name = $2, phone = $3, passport_number = $4, tk = $5, comment = $6
		WHERE id = $7
	`, client.City, client.FullName, client.Phone, client.PassportNumber, client.TK, client.Comment, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client updated successfully"})
}

func DeleteClient(c *gin.Context) {
	id := c.Param("id")

	_, err := DB.Exec(`DELETE FROM clients WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client deleted successfully"})
}
