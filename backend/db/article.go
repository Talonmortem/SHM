package db

import (
	"log"
	"net/http"

	"github.com/Talonmortem/SHM/internal/models"
	"github.com/gin-gonic/gin"
)

func CreateTablesArticles() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS articles (
			service_id BIGSERIAL PRIMARY KEY,
			id BIGINT NOT NULL,
			no INTEGER NOT NULL DEFAULT 0,
			code TEXT NOT NULL,
			description TEXT,
			euro DOUBLE PRECISION DEFAULT 0,
			colli DOUBLE PRECISION DEFAULT 0,
			kg DOUBLE PRECISION DEFAULT 0,
			value DOUBLE PRECISION DEFAULT 0
		);

		ALTER TABLE articles
		ADD COLUMN IF NOT EXISTS service_id BIGINT,
		ADD COLUMN IF NOT EXISTS no INTEGER,
		ADD COLUMN IF NOT EXISTS colli DOUBLE PRECISION,
		ADD COLUMN IF NOT EXISTS kg DOUBLE PRECISION,
		ADD COLUMN IF NOT EXISTS value DOUBLE PRECISION;

		DO $$
		BEGIN
			IF to_regclass('public.articles_service_id_seq') IS NULL THEN
				CREATE SEQUENCE articles_service_id_seq;
			END IF;
		END $$;

		ALTER TABLE articles
		ALTER COLUMN service_id SET DEFAULT nextval('articles_service_id_seq');

		SELECT setval(
			'articles_service_id_seq',
			GREATEST(COALESCE((SELECT MAX(service_id) FROM articles), 0), 1),
			true
		);

		UPDATE articles
		SET service_id = nextval('articles_service_id_seq')
		WHERE service_id IS NULL;

		CREATE UNIQUE INDEX IF NOT EXISTS ux_articles_service_id ON articles(service_id);
		DROP INDEX IF EXISTS ux_articles_id;
		ALTER TABLE articles DROP CONSTRAINT IF EXISTS articles_id_key;

		DO $$
		DECLARE pk_col TEXT;
		BEGIN
			SELECT a.attname
			INTO pk_col
			FROM pg_constraint c
			JOIN pg_class t ON t.oid = c.conrelid
			JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = c.conkey[1]
			WHERE t.relname = 'articles'
				AND c.contype = 'p'
			LIMIT 1;

			IF pk_col IS NULL THEN
				ALTER TABLE articles
				ADD CONSTRAINT articles_pkey PRIMARY KEY (service_id);
			ELSIF pk_col = 'id' THEN
				ALTER TABLE articles DROP CONSTRAINT articles_pkey;
				ALTER TABLE articles
				ADD CONSTRAINT articles_pkey PRIMARY KEY (service_id);
			END IF;
		END $$;

		ALTER TABLE articles
		ALTER COLUMN code TYPE TEXT USING code::text;

		ALTER TABLE articles
		ALTER COLUMN id TYPE BIGINT USING id::BIGINT;

		DO $$
		BEGIN
			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'articles' AND column_name = 'count'
			) THEN
				UPDATE articles SET no = COALESCE(no, count);
				ALTER TABLE articles DROP COLUMN count;
			END IF;

			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'articles' AND column_name = 'weight'
			) THEN
				UPDATE articles SET kg = COALESCE(kg, weight);
				ALTER TABLE articles DROP COLUMN weight;
			END IF;

			IF EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_name = 'articles' AND column_name = 'price'
			) THEN
				UPDATE articles SET value = COALESCE(value, price);
				ALTER TABLE articles DROP COLUMN price;
			END IF;
		END $$;

		UPDATE articles
		SET
			no = COALESCE(no, 0),
			euro = COALESCE(euro, 0),
			colli = COALESCE(colli, 0),
			kg = COALESCE(kg, 0),
			value = COALESCE(value, 0);

		ALTER TABLE articles
		ALTER COLUMN service_id SET NOT NULL,
		ALTER COLUMN no SET NOT NULL,
		ALTER COLUMN no SET DEFAULT 0,
		ALTER COLUMN euro SET DEFAULT 0,
		ALTER COLUMN colli SET DEFAULT 0,
		ALTER COLUMN kg SET DEFAULT 0,
		ALTER COLUMN value SET DEFAULT 0;
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
	if article.ID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required and must be greater than 0"})
		return
	}

	err := DB.QueryRow(`
		INSERT INTO articles (id, no, code, description, euro, colli, kg, value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING service_id
	`, article.ID, article.No, article.Code, article.Description, article.Euro, article.Colli, article.KG, article.Value).Scan(&article.ServiceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, article)
}

func GetArticles(c *gin.Context) {
	rows, err := DB.Query(`SELECT service_id, id, no, code, description, euro, colli, kg, value FROM articles ORDER BY id`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var articles []models.Article
	for rows.Next() {
		var article models.Article
		if err := rows.Scan(&article.ServiceID, &article.ID, &article.No, &article.Code, &article.Description, &article.Euro, &article.Colli, &article.KG, &article.Value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		articles = append(articles, article)
	}

	c.JSON(http.StatusOK, articles)
}

func UpdateArticle(c *gin.Context) {
	serviceID := c.Param("id")
	var article models.Article
	if err := c.ShouldBindJSON(&article); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := DB.Exec(`
		UPDATE articles
		SET id = $1, no = $2, code = $3, description = $4, euro = $5, colli = $6, kg = $7, value = $8
		WHERE service_id = $9
	`, article.ID, article.No, article.Code, article.Description, article.Euro, article.Colli, article.KG, article.Value, serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article updated successfully"})
}

func DeleteArticle(c *gin.Context) {
	serviceID := c.Param("id")

	_, err := DB.Exec(`
		DELETE FROM articles
		WHERE service_id = $1
	`, serviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Article deleted successfully"})
}

func GetBalance(c *gin.Context) {
	rows, err := DB.Query(`
		WITH income AS (
			SELECT
				a.id::BIGINT AS article_id,
				SUM(COALESCE(a.no, 0))::INT AS no,
				MIN(a.code) AS code,
				MIN(a.description) AS description,
				SUM(COALESCE(a.kg, 0)) AS income_kg
			FROM articles a
			GROUP BY a.id
		),
		sent AS (
			SELECT
				a.id::BIGINT AS article_id,
				SUM(
					COALESCE(NULLIF(REPLACE(aip.weight, ',', '.'), '')::DOUBLE PRECISION, 0)
				) AS sent_kg
			FROM article_in_product aip
			INNER JOIN articles a ON a.service_id = aip.article::BIGINT
			INNER JOIN products p ON p.id = aip.product_id
			WHERE p.status = 3
			GROUP BY a.id
		),
		reserved AS (
			SELECT
				a.id::BIGINT AS article_id,
				SUM(
					COALESCE(NULLIF(REPLACE(aip.weight, ',', '.'), '')::DOUBLE PRECISION, 0)
				) AS reserved_kg
			FROM article_in_product aip
			INNER JOIN articles a ON a.service_id = aip.article::BIGINT
			INNER JOIN products p ON p.id = aip.product_id
			WHERE p.status = 2
			GROUP BY a.id
		)
		SELECT
			i.article_id::INT AS id,
			i.no,
			i.code,
			i.description,
			ROUND(COALESCE(i.income_kg, 0)::NUMERIC, 2)::DOUBLE PRECISION AS income_kg,
			ROUND(COALESCE(sent.sent_kg, 0)::NUMERIC, 2)::DOUBLE PRECISION AS sent_kg,
			ROUND((COALESCE(i.income_kg, 0) - COALESCE(sent.sent_kg, 0))::NUMERIC, 2)::DOUBLE PRECISION AS balance_kg,
			ROUND(COALESCE(reserved.reserved_kg, 0)::NUMERIC, 2)::DOUBLE PRECISION AS reserved_kg,
			ROUND((COALESCE(i.income_kg, 0) - COALESCE(sent.sent_kg, 0) - COALESCE(reserved.reserved_kg, 0))::NUMERIC, 2)::DOUBLE PRECISION AS free_kg
		FROM income i
		LEFT JOIN sent ON sent.article_id = i.article_id
		LEFT JOIN reserved ON reserved.article_id = i.article_id
		ORDER BY i.article_id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var balance []models.BalanceRow
	for rows.Next() {
		var row models.BalanceRow
		if err := rows.Scan(
			&row.ID,
			&row.No,
			&row.Code,
			&row.Description,
			&row.IncomeKG,
			&row.SentKG,
			&row.BalanceKG,
			&row.ReservedKG,
			&row.FreeKG,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		balance = append(balance, row)
	}

	c.JSON(http.StatusOK, balance)
}
