package main

import (
	"database/sql"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/Talonmortem/SHM/db"
	_ "modernc.org/sqlite"
)

type counters struct {
	roles            int
	users            int
	articles         int
	products         int
	orders           int
	paymentMethods   int
	payments         int
	articleInProduct int
	orderProducts    int
	rolePermissions  int
	requestLogs      int
}

func main() {
	sqlitePath := strings.TrimSpace(os.Getenv("SQLITE_PATH"))
	if sqlitePath == "" {
		sqlitePath = "../db/inventory.db"
	}

	postgresDSN := strings.TrimSpace(os.Getenv("POSTGRES_DSN"))
	if postgresDSN != "" && strings.TrimSpace(os.Getenv("DB_DSN")) == "" {
		if err := os.Setenv("DB_DSN", postgresDSN); err != nil {
			log.Fatalf("failed to set DB_DSN: %v", err)
		}
	}

	log.Printf("SQLite source: %s", sqlitePath)
	db.ConnectDB()
	defer db.CloseDB()
	db.CreateTables()

	sqliteDB, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		log.Fatalf("failed to open sqlite db: %v", err)
	}
	defer sqliteDB.Close()

	if err := sqliteDB.Ping(); err != nil {
		log.Fatalf("failed to ping sqlite db: %v", err)
	}

	tx, err := db.GetDB().Begin()
	if err != nil {
		log.Fatalf("failed to begin postgres transaction: %v", err)
	}
	defer tx.Rollback()

	if err := truncateTarget(tx); err != nil {
		log.Fatalf("failed to truncate target tables: %v", err)
	}

	var stat counters
	if stat.roles, err = migrateRoles(sqliteDB, tx); err != nil {
		log.Fatalf("roles migration failed: %v", err)
	}
	if stat.users, err = migrateUsers(sqliteDB, tx); err != nil {
		log.Fatalf("users migration failed: %v", err)
	}
	if stat.articles, err = migrateArticles(sqliteDB, tx); err != nil {
		log.Fatalf("articles migration failed: %v", err)
	}
	if stat.products, err = migrateProducts(sqliteDB, tx); err != nil {
		log.Fatalf("products migration failed: %v", err)
	}
	if stat.orders, err = migrateOrders(sqliteDB, tx); err != nil {
		log.Fatalf("orders migration failed: %v", err)
	}
	if stat.paymentMethods, err = migratePaymentMethods(sqliteDB, tx); err != nil {
		log.Fatalf("payment_methods migration failed: %v", err)
	}
	if stat.payments, err = migratePayments(sqliteDB, tx); err != nil {
		log.Fatalf("payments_monitoring migration failed: %v", err)
	}
	if stat.articleInProduct, err = migrateArticleInProduct(sqliteDB, tx); err != nil {
		log.Fatalf("article_in_product migration failed: %v", err)
	}
	if stat.orderProducts, err = migrateOrderProducts(sqliteDB, tx); err != nil {
		log.Fatalf("order_products migration failed: %v", err)
	}
	if stat.rolePermissions, err = migrateRolePermissions(sqliteDB, tx); err != nil {
		log.Fatalf("role_permissions migration failed: %v", err)
	}
	if stat.requestLogs, err = migrateRequestLogs(sqliteDB, tx); err != nil {
		log.Fatalf("request_logs migration failed: %v", err)
	}

	if err := resetSequences(tx); err != nil {
		log.Fatalf("failed to reset sequences: %v", err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatalf("failed to commit migration: %v", err)
	}

	log.Printf("Migration completed successfully")
	log.Printf("roles=%d users=%d articles=%d products=%d orders=%d payment_methods=%d payments=%d article_in_product=%d order_products=%d role_permissions=%d request_logs=%d",
		stat.roles, stat.users, stat.articles, stat.products, stat.orders, stat.paymentMethods, stat.payments, stat.articleInProduct, stat.orderProducts, stat.rolePermissions, stat.requestLogs)
}

func truncateTarget(tx *sql.Tx) error {
	_, err := tx.Exec(`
		TRUNCATE TABLE
			order_products,
			article_in_product,
			payments_monitoring,
			request_logs,
			role_permissions,
			orders,
			products,
			payment_methods,
			users,
			roles,
			articles
		RESTART IDENTITY CASCADE
	`)
	return err
}

func migrateRoles(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`SELECT id, name FROM roles`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var name sql.NullString
		if err := rows.Scan(&id, &name); err != nil {
			return n, err
		}
		if _, err := dst.Exec(`INSERT INTO roles (id, name) VALUES ($1, $2)`, id, name); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migrateUsers(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`
		SELECT
			id,
			username,
			password,
			CASE
				WHEN role_id IS NULL OR TRIM(role_id) = '' THEN NULL
				ELSE CAST(role_id AS INTEGER)
			END AS role_id
		FROM users
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var username, password sql.NullString
		var roleID sql.NullInt64
		if err := rows.Scan(&id, &username, &password, &roleID); err != nil {
			return n, err
		}
		if _, err := dst.Exec(`INSERT INTO users (id, username, password, role_id) VALUES ($1, $2, $3, $4)`, id, username, password, roleID); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migrateArticles(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`SELECT id, code, description, euro, count, weight, price FROM articles`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var code sql.NullInt64
		var description sql.NullString
		var euro, weight, price sql.NullFloat64
		var count sql.NullInt64
		if err := rows.Scan(&id, &code, &description, &euro, &count, &weight, &price); err != nil {
			return n, err
		}
		var codeValue any
		if code.Valid {
			codeValue = strconv.FormatInt(code.Int64, 10)
		}
		if _, err := dst.Exec(`
			INSERT INTO articles (id, code, description, euro, count, weight, price)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, id, codeValue, description, euro, count, weight, price); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migrateProducts(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`
		SELECT id, status, name, article, weight, skidka, summaRubSoSkidkoj, count, onePrice, video, description
		FROM products
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var status sql.NullInt64
		var name, weight, skidka, summa, onePrice, video, description sql.NullString
		var article, count sql.NullInt64
		if err := rows.Scan(&id, &status, &name, &article, &weight, &skidka, &summa, &count, &onePrice, &video, &description); err != nil {
			return n, err
		}
		if _, err := dst.Exec(`
			INSERT INTO products (id, status, name, article, weight, skidka, summaRubSoSkidkoj, count, onePrice, video, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, id, status, name, article, weight, skidka, summa, count, onePrice, video, description); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migrateOrders(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`SELECT id, name, quantity, status, description, debt FROM orders`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var name, description sql.NullString
		var quantity, status sql.NullInt64
		var debt sql.NullFloat64
		if err := rows.Scan(&id, &name, &quantity, &status, &description, &debt); err != nil {
			return n, err
		}
		if _, err := dst.Exec(`
			INSERT INTO orders (id, name, quantity, status, description, debt)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, id, name, quantity, status, description, debt); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migratePaymentMethods(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`SELECT id, method FROM payment_methods`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var method sql.NullString
		if err := rows.Scan(&id, &method); err != nil {
			return n, err
		}
		if _, err := dst.Exec(`INSERT INTO payment_methods (id, method) VALUES ($1, $2)`, id, method); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migratePayments(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`SELECT id, date, method, order_id, amount, comment FROM payments_monitoring`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var date, method, comment sql.NullString
		var orderID sql.NullInt64
		var amount sql.NullFloat64
		if err := rows.Scan(&id, &date, &method, &orderID, &amount, &comment); err != nil {
			return n, err
		}
		if _, err := dst.Exec(`
			INSERT INTO payments_monitoring (id, date, method, order_id, amount, comment)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, id, date, method, orderID, amount, comment); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migrateArticleInProduct(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`SELECT id, product_id, article, cursEvro, priceEvro, weight, count, sumEvro, sumRub FROM article_in_product`)
	withCount := err == nil
	if err != nil {
		rows, err = src.Query(`SELECT id, product_id, article, cursEvro, priceEvro, weight, sumEvro, sumRub FROM article_in_product`)
		if err != nil {
			return 0, err
		}
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var productID, article sql.NullInt64
		var count sql.NullInt64
		var cursEvro, priceEvro, weight, sumEvro, sumRub sql.NullString
		if withCount {
			if err := rows.Scan(&id, &productID, &article, &cursEvro, &priceEvro, &weight, &count, &sumEvro, &sumRub); err != nil {
				return n, err
			}
		} else {
			if err := rows.Scan(&id, &productID, &article, &cursEvro, &priceEvro, &weight, &sumEvro, &sumRub); err != nil {
				return n, err
			}
		}
		if _, err := dst.Exec(`
			INSERT INTO article_in_product (id, product_id, article, cursEvro, priceEvro, weight, count, sumEvro, sumRub)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, id, productID, article, cursEvro, priceEvro, weight, count, sumEvro, sumRub); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migrateOrderProducts(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`SELECT id, order_id, product_id FROM order_products`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var orderID, productID sql.NullInt64
		if err := rows.Scan(&id, &orderID, &productID); err != nil {
			return n, err
		}
		if _, err := dst.Exec(`INSERT INTO order_products (id, order_id, product_id) VALUES ($1, $2, $3)`, id, orderID, productID); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migrateRolePermissions(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`SELECT id, role_id, method, path, allowed FROM role_permissions`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var roleID sql.NullInt64
		var method, path sql.NullString
		var allowed sql.NullBool
		if err := rows.Scan(&id, &roleID, &method, &path, &allowed); err != nil {
			return n, err
		}
		if _, err := dst.Exec(`
			INSERT INTO role_permissions (id, role_id, method, path, allowed)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (role_id, method, path) DO UPDATE
			SET allowed = EXCLUDED.allowed
		`, id, roleID, method, path, allowed); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func migrateRequestLogs(src *sql.DB, dst *sql.Tx) (int, error) {
	rows, err := src.Query(`SELECT id, username, method, path, params, created_at FROM request_logs`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	n := 0
	for rows.Next() {
		var id int
		var username, method, path, params, createdAt sql.NullString
		if err := rows.Scan(&id, &username, &method, &path, &params, &createdAt); err != nil {
			return n, err
		}
		if _, err := dst.Exec(`
			INSERT INTO request_logs (id, username, method, path, params, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, id, username, method, path, params, createdAt); err != nil {
			return n, err
		}
		n++
	}
	return n, rows.Err()
}

func resetSequences(tx *sql.Tx) error {
	tables := []string{
		"roles",
		"users",
		"articles",
		"products",
		"orders",
		"payment_methods",
		"payments_monitoring",
		"article_in_product",
		"order_products",
		"role_permissions",
		"request_logs",
	}

	for _, table := range tables {
		if _, err := tx.Exec(`
			SELECT setval(
				pg_get_serial_sequence($1, 'id'),
				COALESCE((SELECT MAX(id) FROM `+table+`), 1),
				COALESCE((SELECT MAX(id) FROM `+table+`), 0) > 0
			)
		`, table); err != nil {
			return err
		}
	}

	return nil
}
