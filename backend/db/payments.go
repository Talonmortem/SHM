//module for payments monitoring

package db

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Talonmortem/SHM/internal/models"
	"github.com/gin-gonic/gin"
)

func GetPaymentsMonitoring(c *gin.Context) {
	method := c.Query("method")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	base := `SELECT pm.id, pm.date, pm.method, pm.amount, pm.comment
	FROM payments_monitoring pm
	JOIN payment_methods pp ON pp.method = pm.method`

	args := []any{}
	conditions := []string{}

	if method != "" {
		args = append(args, method)
		conditions = append(conditions, fmt.Sprintf("pp.method = $%d", len(args)))
	}

	if dateFrom != "" && dateTo != "" {
		from := dateFrom
		to := dateTo
		if len(strings.TrimSpace(from)) == len("2006-01-02") {
			from += " 00:00:00"
		}
		if len(strings.TrimSpace(to)) == len("2006-01-02") {
			to += " 23:59:59"
		}
		args = append(args, from)
		args = append(args, to)
		conditions = append(conditions, fmt.Sprintf("pm.date BETWEEN $%d AND $%d", len(args)-1, len(args)))
	} else if dateFrom != "" {
		from := dateFrom
		if len(strings.TrimSpace(from)) == len("2006-01-02") {
			from += " 00:00:00"
		}
		args = append(args, from)
		conditions = append(conditions, fmt.Sprintf("pm.date >= $%d", len(args)))
	} else if dateTo != "" {
		to := dateTo
		if len(strings.TrimSpace(to)) == len("2006-01-02") {
			to += " 23:59:59"
		}
		args = append(args, to)
		conditions = append(conditions, fmt.Sprintf("pm.date <= $%d", len(args)))
	}

	query := base
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY pm.date"

	row, err := DB.Query(query, args...)
	if err != nil {
		log.Printf("Error querying payments monitoring: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve payments monitoring data"})
		return
	}
	defer row.Close()

	var payments []models.Payment
	for row.Next() {
		var p models.Payment
		if err := row.Scan(&p.ID, &p.Date, &p.Method, &p.Amount, &p.Comment); err != nil {
			log.Printf("Error scanning row: %v\n", err)
			continue
		}
		payments = append(payments, p)
	}

	c.JSON(http.StatusOK, payments)
}

func GetPaymentMethods(c *gin.Context) {
	rows, err := DB.Query("SELECT id, method FROM payment_methods ORDER BY method")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var paymentMethods []struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
	}
	for rows.Next() {
		var pm struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		if err := rows.Scan(&pm.ID, &pm.Method); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		paymentMethods = append(paymentMethods, pm)
	}
	c.JSON(http.StatusOK, paymentMethods)
}

func CreatePayment(c *gin.Context) {
	var payment models.Payment
	if err := c.ShouldBindJSON(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	normalizedDate, err := normalizePaymentDateInput(payment.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment date format"})
		return
	}

	query := `INSERT INTO payments_monitoring (date, method, amount, comment)
				VALUES ($1, $2, $3, $4)`
	_, err = DB.Exec(query, normalizedDate, payment.Method, payment.Amount, payment.Comment)
	if err != nil {
		log.Printf("Error inserting payment: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Payment created successfully"})
}

func UpdatePayment(c *gin.Context) {
	id := c.Param("id")
	var payment models.Payment
	if err := c.ShouldBindJSON(&payment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	normalizedDate, err := normalizePaymentDateInput(payment.Date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment date format"})
		return
	}

	query := `UPDATE payments_monitoring SET date = $1, method = $2, amount = $3, comment = $4
				WHERE id = $5`
	_, err = DB.Exec(query, normalizedDate, payment.Method, payment.Amount, payment.Comment, id)
	if err != nil {
		log.Printf("Error updating payment: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment updated successfully"})
}

func DeletePayment(c *gin.Context) {
	id := c.Param("id")

	_, err := DB.Exec("DELETE FROM payments_monitoring WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting payment: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete payment"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment deleted successfully"})
}
