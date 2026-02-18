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

		ALTER TABLE article_in_product
		ALTER COLUMN article TYPE BIGINT USING article::BIGINT;

		UPDATE article_in_product aip
		SET article = a.service_id
		FROM articles a
		WHERE aip.article = a.id
			AND NOT EXISTS (
				SELECT 1 FROM articles ax WHERE ax.service_id = aip.article
			);

		UPDATE article_in_product aip
		SET article = mapped.service_id
		FROM (
			SELECT code, MIN(service_id) AS service_id
			FROM articles
			GROUP BY code
		) mapped
		WHERE aip.article::TEXT = mapped.code
			AND NOT EXISTS (
				SELECT 1 FROM articles ax WHERE ax.service_id = aip.article
			);

		ALTER TABLE article_in_product
		DROP CONSTRAINT IF EXISTS fk_article_in_product_article_id;

		DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_constraint
				WHERE conname = 'fk_article_in_product_article_id'
			) THEN
				ALTER TABLE article_in_product
				ADD CONSTRAINT fk_article_in_product_article_id
				FOREIGN KEY (article) REFERENCES articles(service_id)
				ON UPDATE CASCADE
				ON DELETE RESTRICT
				NOT VALID;
			END IF;
		END $$;

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
	totalWeight := 0.0

	for _, a := range p.ArticlesInProduct {
		totalSumRub += parseNumericInput(a.SumRub)
		totalWeight += parseNumericInput(a.Weight)
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
	p.Weight = strconv.FormatFloat(totalWeight, 'f', 2, 64)

	onePrice := 0.0
	if totalWeight != 0 {
		onePrice = discountedSum / totalWeight
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
		calculateArticleFields(&product.ArticlesInProduct[i])
	}

	return nil
}

func collectArticleWeights(articlesInProduct []models.ArticleInProduct) map[int]float64 {
	weights := make(map[int]float64)
	for _, a := range articlesInProduct {
		if a.Article <= 0 {
			continue
		}
		weights[a.Article] += parseNumericInput(a.Weight)
	}
	return weights
}

func reserveArticleStock(tx *sql.Tx, requestedWeights map[int]float64) error {
	for articleServiceID, requested := range requestedWeights {
		var existingKG float64
		err := tx.QueryRow("SELECT COALESCE(kg, 0) FROM articles WHERE service_id = $1 FOR UPDATE", articleServiceID).Scan(&existingKG)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return newBadRequestError(fmt.Sprintf("Article %d does not exist", articleServiceID))
			}
			return err
		}
		if _, err := tx.Exec("UPDATE articles SET kg = COALESCE(kg, 0) - $1 WHERE service_id = $2", requested, articleServiceID); err != nil {
			return err
		}
	}

	return nil
}

func releaseArticleStock(tx *sql.Tx, releasedWeights map[int]float64) error {
	for articleServiceID, released := range releasedWeights {
		if released == 0 {
			continue
		}

		result, err := tx.Exec("UPDATE articles SET kg = COALESCE(kg, 0) + $1 WHERE service_id = $2", released, articleServiceID)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return fmt.Errorf("article %d not found while restoring stock", articleServiceID)
		}
	}

	return nil
}

func getReservedArticleWeightsByProductID(tx *sql.Tx, productID int) (map[int]float64, error) {
	rows, err := tx.Query("SELECT article, weight FROM article_in_product WHERE product_id = $1", productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	weights := make(map[int]float64)
	for rows.Next() {
		var articleCode int
		var weightRaw string
		if err := rows.Scan(&articleCode, &weightRaw); err != nil {
			return nil, err
		}
		if articleCode <= 0 {
			continue
		}
		weights[articleCode] += parseNumericInput(weightRaw)
	}

	return weights, rows.Err()
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
	rows, err := DB.Query("SELECT id, status, name, weight, skidka, summaRubSoSkidkoj, count, onePrice, video, description FROM products")
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
			&product.Weight,
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

	requestedWeights := collectArticleWeights(product.ArticlesInProduct)
	if err := reserveArticleStock(tx, requestedWeights); err != nil {
		writeProductError(c, err)
		return
	}

	calculateProductFields(&product)

	err = tx.QueryRow(
		"INSERT INTO products (status, name, weight, skidka, summaRubSoSkidkoj, count, onePrice, video, description) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id",
		product.Status,
		product.Name,
		product.Weight,
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

	oldCounts, err := getReservedArticleWeightsByProductID(tx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := releaseArticleStock(tx, oldCounts); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	newCounts := collectArticleWeights(product.ArticlesInProduct)
	if err := reserveArticleStock(tx, newCounts); err != nil {
		writeProductError(c, err)
		return
	}

	calculateProductFields(&product)

	result, err := tx.Exec(
		"UPDATE products SET status = $1, name = $2, weight = $3, skidka = $4, summaRubSoSkidkoj = $5, count = $6, onePrice = $7, video = $8, description = $9 WHERE id = $10",
		product.Status,
		product.Name,
		product.Weight,
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

	oldCounts, err := getReservedArticleWeightsByProductID(tx, id)
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
