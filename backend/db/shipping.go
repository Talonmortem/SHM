package db

import (
	"log"
	"net/http"
	"strings"

	"github.com/Talonmortem/SHM/internal/models"
	"github.com/gin-gonic/gin"
)

func CreateTablesShipping() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS shipments (
			id BIGSERIAL PRIMARY KEY,
			ship_date TEXT NOT NULL,
			city TEXT,
			full_name TEXT NOT NULL,
			phone TEXT,
			passport_inn TEXT,
			tk TEXT,
			places INTEGER,
			price DOUBLE PRECISION,
			weight DOUBLE PRECISION
		);

		CREATE TABLE IF NOT EXISTS shipment_notes (
			id BIGSERIAL PRIMARY KEY,
			ship_date TEXT NOT NULL,
			note TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX IF NOT EXISTS idx_shipments_ship_date ON shipments(ship_date);
		CREATE INDEX IF NOT EXISTS idx_shipments_city ON shipments(city);
		CREATE INDEX IF NOT EXISTS idx_shipments_full_name ON shipments(full_name);
		CREATE INDEX IF NOT EXISTS idx_shipment_notes_ship_date ON shipment_notes(ship_date);
	`)
	if err != nil {
		log.Fatal("Failed to create shipments tables:", err)
	}
	log.Println("Shipments tables are ready")
}

func CreateShipment(c *gin.Context) {
	var shipment models.Shipment
	if err := c.ShouldBindJSON(&shipment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(shipment.ShipDate) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Дата отправки обязательна"})
		return
	}
	if strings.TrimSpace(shipment.FullName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО обязательно"})
		return
	}

	err := DB.QueryRow(`
		INSERT INTO shipments (ship_date, city, full_name, phone, passport_inn, tk, places, price, weight)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`, shipment.ShipDate, shipment.City, shipment.FullName, shipment.Phone, shipment.PassportInn, shipment.TK, shipment.Places, shipment.Price, shipment.Weight).Scan(&shipment.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, shipment)
}

func GetShipments(c *gin.Context) {
	date := strings.TrimSpace(c.Query("date"))

	query := `
		SELECT id, ship_date, city, full_name, phone, passport_inn, tk, places, price, weight
		FROM shipments
	`
	args := []any{}
	if date != "" {
		query += " WHERE ship_date = $1"
		args = append(args, date)
	}
	query += " ORDER BY ship_date DESC, id DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	shipments := make([]models.Shipment, 0)
	for rows.Next() {
		var shipment models.Shipment
		if err := rows.Scan(
			&shipment.ID,
			&shipment.ShipDate,
			&shipment.City,
			&shipment.FullName,
			&shipment.Phone,
			&shipment.PassportInn,
			&shipment.TK,
			&shipment.Places,
			&shipment.Price,
			&shipment.Weight,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		shipments = append(shipments, shipment)
	}

	c.JSON(http.StatusOK, shipments)
}

func UpdateShipment(c *gin.Context) {
	id := c.Param("id")
	var shipment models.Shipment
	if err := c.ShouldBindJSON(&shipment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(shipment.ShipDate) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Дата отправки обязательна"})
		return
	}
	if strings.TrimSpace(shipment.FullName) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ФИО обязательно"})
		return
	}

	_, err := DB.Exec(`
		UPDATE shipments
		SET ship_date = $1, city = $2, full_name = $3, phone = $4, passport_inn = $5, tk = $6, places = $7, price = $8, weight = $9
		WHERE id = $10
	`, shipment.ShipDate, shipment.City, shipment.FullName, shipment.Phone, shipment.PassportInn, shipment.TK, shipment.Places, shipment.Price, shipment.Weight, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Shipment updated successfully"})
}

func DeleteShipment(c *gin.Context) {
	id := c.Param("id")

	_, err := DB.Exec(`DELETE FROM shipments WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Shipment deleted successfully"})
}

func CreateShipmentNote(c *gin.Context) {
	var note models.ShipmentNote
	if err := c.ShouldBindJSON(&note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(note.ShipDate) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Дата обязательна"})
		return
	}
	if strings.TrimSpace(note.Note) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Заметка обязательна"})
		return
	}

	err := DB.QueryRow(`
		INSERT INTO shipment_notes (ship_date, note)
		VALUES ($1, $2)
		RETURNING id
	`, note.ShipDate, note.Note).Scan(&note.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, note)
}

func GetShipmentNotes(c *gin.Context) {
	date := strings.TrimSpace(c.Query("date"))

	query := `SELECT id, ship_date, note FROM shipment_notes`
	args := []any{}
	if date != "" {
		query += " WHERE ship_date = $1"
		args = append(args, date)
	}
	query += " ORDER BY ship_date DESC, id DESC"

	rows, err := DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	notes := make([]models.ShipmentNote, 0)
	for rows.Next() {
		var note models.ShipmentNote
		if err := rows.Scan(&note.ID, &note.ShipDate, &note.Note); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		notes = append(notes, note)
	}

	c.JSON(http.StatusOK, notes)
}

func UpdateShipmentNote(c *gin.Context) {
	id := c.Param("id")
	var note models.ShipmentNote
	if err := c.ShouldBindJSON(&note); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(note.ShipDate) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Дата обязательна"})
		return
	}
	if strings.TrimSpace(note.Note) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Заметка обязательна"})
		return
	}

	_, err := DB.Exec(`
		UPDATE shipment_notes
		SET ship_date = $1, note = $2
		WHERE id = $3
	`, note.ShipDate, note.Note, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Shipment note updated successfully"})
}

func DeleteShipmentNote(c *gin.Context) {
	id := c.Param("id")

	_, err := DB.Exec(`DELETE FROM shipment_notes WHERE id = $1`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Shipment note deleted successfully"})
}
