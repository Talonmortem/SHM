package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Talonmortem/SHM/internal/models"
	"github.com/gin-gonic/gin"
)

type badRequestError struct {
	message string
}

func (e *badRequestError) Error() string {
	return e.message
}

func newBadRequestError(message string) error {
	return &badRequestError{message: message}
}

func writeProductError(c *gin.Context, err error) {
	var badReq *badRequestError
	if errors.As(err, &badReq) {
		c.JSON(http.StatusBadRequest, gin.H{"error": badReq.message})
		return
	}

	log.Printf("product handler error: %v", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

func CreateTablesProducts() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS products (
			id BIGSERIAL PRIMARY KEY,
			status INTEGER NOT NULL,
			name TEXT NOT NULL,
			article INTEGER,
			weight TEXT,
			skidka TEXT,
			summaRubSoSkidkoj TEXT,
			count INTEGER,
			onePrice TEXT,
			video TEXT,
			description TEXT
		);

		CREATE TABLE IF NOT EXISTS article_in_product (
			id BIGSERIAL PRIMARY KEY,
			product_id BIGINT NOT NULL,
			article INTEGER NOT NULL,
			cursEvro TEXT,
			priceEvro TEXT,
			weight TEXT,
			count INTEGER NOT NULL DEFAULT 0,
			sumEvro TEXT,
			sumRub TEXT,
			FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
		);

		ALTER TABLE article_in_product
		ADD COLUMN IF NOT EXISTS count INTEGER NOT NULL DEFAULT 0;

		SELECT setval(
			pg_get_serial_sequence('products', 'id'),
			GREATEST((SELECT COALESCE(MAX(id), 0) FROM products), 5999),
			true
		);
	`)
	if err != nil {
		log.Fatal("Failed to create products table:", err)
	}
	log.Println("Products table created successfully!")
}

func calculateArticleFields(a *models.ArticleInProduct) {
	priceEvro := parseNumericInput(a.PriceEvro)
	cursEvro := parseNumericInput(a.CursEvro)
	weight := parseNumericInput(a.Weight)

	sumEvro := priceEvro * weight
	sumRub := sumEvro * cursEvro

	a.SumEvro = strconv.FormatFloat(sumEvro, 'f', 2, 64)
	a.SumRub = strconv.FormatFloat(sumRub, 'f', 2, 64)
}

func parseNumericInput(raw string) float64 {
	clean := strings.TrimSpace(raw)
	if clean == "" {
		return 0
	}
	clean = strings.ReplaceAll(clean, "%", "")
	clean = strings.ReplaceAll(clean, ",", ".")
	v, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0
	}
	return v
}

func calculateProductFields(p *models.Product) {
	var totalSumRub float64
	totalCount := 0

	for _, a := range p.ArticlesInProduct {
		totalSumRub += parseNumericInput(a.SumRub)
		totalCount += a.Count
	}

	skidkaPercent := parseNumericInput(p.Skidka)
	if skidkaPercent < 0 {
		skidkaPercent = 0
	}
	if skidkaPercent > 100 {
		skidkaPercent = 100
	}

	discountedSum := totalSumRub * (1 - skidkaPercent/100)
	p.SummaRubSoSkidkoj = strconv.FormatFloat(discountedSum, 'f', 2, 64)
	p.Count = totalCount

	onePrice := 0.0
	if p.Count > 0 {
		onePrice = discountedSum / float64(p.Count)
	}
	p.OnePrice = strconv.FormatFloat(onePrice, 'f', 2, 64)
}

func validateAndPrepareProduct(product *models.Product) error {
	if product.Status < 1 || product.Status > 3 {
		return newBadRequestError("Status must be 1 (На продаже), 2 (Забронировано), or 3 (Продано)")
	}

	for i := range product.ArticlesInProduct {
		if product.ArticlesInProduct[i].Article <= 0 {
			return newBadRequestError("Article code is required in Articles in Product")
		}
		if product.ArticlesInProduct[i].Count <= 0 {
			return newBadRequestError(fmt.Sprintf("Count for article %d must be greater than 0", product.ArticlesInProduct[i].Article))
		}
		calculateArticleFields(&product.ArticlesInProduct[i])
	}

	return nil
}

func collectArticleCounts(articlesInProduct []models.ArticleInProduct) map[int]int {
	counts := make(map[int]int)
	for _, a := range articlesInProduct {
		if a.Article <= 0 || a.Count <= 0 {
			continue
		}
		counts[a.Article] += a.Count
	}
	return counts
}

func reserveArticleStock(tx *sql.Tx, requestedCounts map[int]int) error {
	for articleCode, requested := range requestedCounts {
		var available int
		err := tx.QueryRow("SELECT count FROM articles WHERE code::integer = $1 FOR UPDATE", articleCode).Scan(&available)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return newBadRequestError(fmt.Sprintf("Article %d does not exist", articleCode))
			}
			return err
		}
		if available < requested {
			return newBadRequestError(fmt.Sprintf("Not enough stock for article %d: available %d, requested %d", articleCode, available, requested))
		}
		if _, err := tx.Exec("UPDATE articles SET count = count - $1 WHERE code::integer = $2", requested, articleCode); err != nil {
			return err
		}
	}

	return nil
}

func releaseArticleStock(tx *sql.Tx, releasedCounts map[int]int) error {
	for articleCode, released := range releasedCounts {
		if released <= 0 {
			continue
		}

		result, err := tx.Exec("UPDATE articles SET count = count + $1 WHERE code::integer = $2", released, articleCode)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf("article %d not found while restoring stock", articleCode)
		}
	}

	return nil
}

func getReservedArticleCountsByProductID(tx *sql.Tx, productID int) (map[int]int, error) {
	rows, err := tx.Query("SELECT article, count FROM article_in_product WHERE product_id = $1", productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[int]int)
	for rows.Next() {
		var articleCode int
		var count int
		if err := rows.Scan(&articleCode, &count); err != nil {
			return nil, err
		}
		if articleCode <= 0 || count <= 0 {
			continue
		}
		counts[articleCode] += count
	}

	return counts, rows.Err()
}

func insertProductArticles(tx *sql.Tx, productID int, articlesInProduct []models.ArticleInProduct) error {
	for i := range articlesInProduct {
		err := tx.QueryRow(
			"INSERT INTO article_in_product (product_id, article, cursEvro, priceEvro, weight, count, sumEvro, sumRub) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id",
			productID,
			articlesInProduct[i].Article,
			articlesInProduct[i].CursEvro,
			articlesInProduct[i].PriceEvro,
			articlesInProduct[i].Weight,
			articlesInProduct[i].Count,
			articlesInProduct[i].SumEvro,
			articlesInProduct[i].SumRub,
		).Scan(&articlesInProduct[i].ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func GetProducts(c *gin.Context) {
	rows, err := DB.Query("SELECT id, status, name, skidka, summaRubSoSkidkoj, count, onePrice, video, description FROM products")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "articles": "Make sure articles exist in articles table"})
		return
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		if err := rows.Scan(
			&product.ID,
			&product.Status,
			&product.Name,
			&product.Skidka,
			&product.SummaRubSoSkidkoj,
			&product.Count,
			&product.OnePrice,
			&product.Video,
			&product.Description,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		articleRows, err := DB.Query("SELECT id, article, cursEvro, priceEvro, weight, count, sumEvro, sumRub FROM article_in_product WHERE product_id = $1", product.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		for articleRows.Next() {
			var article models.ArticleInProduct
			if err := articleRows.Scan(&article.ID, &article.Article, &article.CursEvro, &article.PriceEvro, &article.Weight, &article.Count, &article.SumEvro, &article.SumRub); err != nil {
				articleRows.Close()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			product.ArticlesInProduct = append(product.ArticlesInProduct, article)
		}
		articleRows.Close()
		products = append(products, product)
	}

	c.JSON(http.StatusOK, products)
}

func CreateProduct(c *gin.Context) {
	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateAndPrepareProduct(&product); err != nil {
		writeProductError(c, err)
		return
	}

	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	requestedCounts := collectArticleCounts(product.ArticlesInProduct)
	if err := reserveArticleStock(tx, requestedCounts); err != nil {
		writeProductError(c, err)
		return
	}

	calculateProductFields(&product)

	err = tx.QueryRow(
		"INSERT INTO products (status, name, skidka, summaRubSoSkidkoj, count, onePrice, video, description) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id",
		product.Status,
		product.Name,
		product.Skidka,
		product.SummaRubSoSkidkoj,
		product.Count,
		product.OnePrice,
		product.Video,
		product.Description,
	).Scan(&product.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := insertProductArticles(tx, product.ID, product.ArticlesInProduct); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, product)
}

func UpdateProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var product models.Product
	if err := c.ShouldBindJSON(&product); err != nil {
		log.Printf("Error binding JSON for product update: %v \n %v", product, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validateAndPrepareProduct(&product); err != nil {
		writeProductError(c, err)
		return
	}

	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	oldCounts, err := getReservedArticleCountsByProductID(tx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := releaseArticleStock(tx, oldCounts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newCounts := collectArticleCounts(product.ArticlesInProduct)
	if err := reserveArticleStock(tx, newCounts); err != nil {
		writeProductError(c, err)
		return
	}

	calculateProductFields(&product)

	result, err := tx.Exec(
		"UPDATE products SET status = $1, name = $2, skidka = $3, summaRubSoSkidkoj = $4, count = $5, onePrice = $6, video = $7, description = $8 WHERE id = $9",
		product.Status,
		product.Name,
		product.Skidka,
		product.SummaRubSoSkidkoj,
		product.Count,
		product.OnePrice,
		product.Video,
		product.Description,
		id,
	)
	if err != nil {
		log.Printf("Error updating product: %v \n %v", product, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error updating product": err.Error()})
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	_, err = tx.Exec("DELETE FROM article_in_product WHERE product_id = $1", id)
	if err != nil {
		log.Printf("Error deleting old articles for product: %v \n %v", product, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error deleting product": err.Error()})
		return
	}

	if err := insertProductArticles(tx, id, product.ArticlesInProduct); err != nil {
		log.Printf("Error inserting product articles for product %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error inserting article": err.Error()})
		return
	}

	orderRows, err := tx.Query("SELECT DISTINCT order_id FROM order_products WHERE product_id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch related orders: " + err.Error()})
		return
	}

	var relatedOrderIDs []int
	for orderRows.Next() {
		var orderID int
		if err := orderRows.Scan(&orderID); err != nil {
			orderRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse related orders: " + err.Error()})
			return
		}
		relatedOrderIDs = append(relatedOrderIDs, orderID)
	}
	if err := orderRows.Err(); err != nil {
		orderRows.Close()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read related orders: " + err.Error()})
		return
	}
	orderRows.Close()

	for _, orderID := range relatedOrderIDs {
		if err := recalculateOrderDebt(tx, orderID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to recalculate order debt: " + err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	product.ID = id
	c.JSON(http.StatusOK, product)
}

func DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	oldCounts, err := getReservedArticleCountsByProductID(tx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := releaseArticleStock(tx, oldCounts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result, err := tx.Exec("DELETE FROM products WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	affected, err := result.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted"})
}
