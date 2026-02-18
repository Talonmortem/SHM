package db

import (
	"log"
	"net/http"
	"strconv"

	"github.com/Talonmortem/SHM/internal/models"
	"github.com/gin-gonic/gin"
)

func CreateTablesArticles() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			id BIGSERIAL PRIMARY KEY,
			code VARCHAR(100) NOT NULL,
			description TEXT,
			euro NUMERIC(10,2),
			count INTEGER,
			weight NUMERIC(10,2),
			price NUMERIC(10,2)
		)
	`)
	if err != nil {
		log.Fatal("Failed to create articles table:", err)
	}
	log.Println("Articles table is ready")
}

func CreateArticle(c *gin.Context) {
	var article models.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := DB.QueryRow(`
		INSERT INTO articles (code, description, euro, count, weight, price)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, strconv.Itoa(article.Code), article.Description, article.Euro, article.Count, article.Weight, article.Price).Scan(&article.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, article)
}

func GetArticles(c *gin.Context) {
	rows, err := DB.Query(`SELECT id, code, description, euro, count, weight, price FROM articles`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		if err := rows.Scan(&article.ID, &article.Code, &article.Description, &article.Euro, &article.Count, &article.Weight, &article.Price); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		articles = append(articles, article)
	}

	c.JSON(http.StatusOK, articles)
}

func UpdateArticle(c *gin.Context) {
	id := c.Param("id")
	var article models.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`
		UPDATE articles
		SET code = $1, description = $2, euro = $3, count = $4, weight = $5, price = $6
		WHERE id = $7
	`, strconv.Itoa(article.Code), article.Description, article.Euro, article.Count, article.Weight, article.Price, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article updated successfully"})
}

func DeleteArticle(c *gin.Context) {
	id := c.Param("id")

	_, err := DB.Exec(`
		DELETE FROM articles
		WHERE id = $1
	`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article deleted successfully"})
}
