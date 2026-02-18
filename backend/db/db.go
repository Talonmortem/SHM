package db

import (
	"database/sql"
	"log"
	"os"
	"strings"

	"github.com/Talonmortem/SHM/internal/models"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

// Logging
func SaveRequestLog(logEntry *models.RequestLog) error {
	_, err := DB.Exec(`
		INSERT INTO request_logs (username, method, path, params, created_at)
		VALUES ($1, $2, $3, $4, $5)`,
		logEntry.Username, logEntry.Method, logEntry.Path, logEntry.Params, logEntry.CreatedAt,
	)
	return err
}

func resolveDSN() string {
	if envDSN := strings.TrimSpace(os.Getenv("DB_DSN")); envDSN != "" {
		return envDSN
	}

	return "postgres://postgres:postgres@localhost:5432/shm?sslmode=disable"
}

func ConnectDB() {
	dsn := resolveDSN()
	var err error
	DB, err = sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}

	err = DB.Ping()
	if err != nil {
		log.Fatal("Database connection failed:", err)
	}
	log.Printf("Database connected")
}

func GetDB() *sql.DB {
	return DB
}

func CloseDB() {
	if DB != nil {
		err := DB.Close()
		if err != nil {
			log.Println("Failed to close database:", err)
		} else {
			log.Println("Database closed successfully.")
		}
	}
}

func CreateTables() {
	CreateTablesArticles()
	CreateTablesProducts()
	CreateTablesUsers()
	CreateTablesClients()
	CreateTablesShipping()

	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			id BIGSERIAL PRIMARY KEY,
			name TEXT,
			quantity INTEGER NOT NULL,
			status INTEGER NOT NULL,
			description TEXT,
			debt DOUBLE PRECISION DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS order_products (
			id BIGSERIAL PRIMARY KEY,
			order_id BIGINT,
			product_id BIGINT,
			FOREIGN KEY(order_id) REFERENCES orders(id) ON DELETE CASCADE,
			FOREIGN KEY(product_id) REFERENCES products(id) ON DELETE CASCADE
		);

		CREATE TABLE IF NOT EXISTS payment_methods (
			id BIGSERIAL PRIMARY KEY,
			method TEXT UNIQUE
		);

		CREATE TABLE IF NOT EXISTS payments_monitoring (
			id BIGSERIAL PRIMARY KEY,
			date TEXT,
			method TEXT,
			order_id BIGINT,
			amount DOUBLE PRECISION,
			comment TEXT,
			FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE SET NULL
		);

		CREATE TABLE IF NOT EXISTS request_logs (
			id BIGSERIAL PRIMARY KEY,
			username TEXT,
			method TEXT,
			path TEXT,
			params TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}
	log.Println("Tables created successfully!")

	_, err = DB.Exec(`
		UPDATE payments_monitoring
		SET date = date || ' 00:00:00'
		WHERE date ~ '^\d{4}-\d{2}-\d{2}$';

		UPDATE payments_monitoring
		SET date = SUBSTRING(date, 7, 4) || '-' || SUBSTRING(date, 4, 2) || '-' || SUBSTRING(date, 1, 2) || ' 00:00:00'
		WHERE date ~ '^\d{2}-\d{2}-\d{4}$';

		UPDATE payments_monitoring
		SET date = SUBSTRING(date, 7, 4) || '-' || SUBSTRING(date, 4, 2) || '-' || SUBSTRING(date, 1, 2) || SUBSTRING(date, 11)
		WHERE date ~ '^\d{2}-\d{2}-\d{4}\s+\d{2}:\d{2}:\d{2}$';

		DELETE FROM payments_monitoring p1
		USING payments_monitoring p2
		WHERE p1.id > p2.id
			AND p1.order_id IS NOT NULL
			AND p2.order_id IS NOT NULL
			AND p1.order_id = p2.order_id
			AND COALESCE(p1.method, '') = COALESCE(p2.method, '')
			AND COALESCE(p1.amount, 0) = COALESCE(p2.amount, 0)
			AND COALESCE(p1.comment, '') = COALESCE(p2.comment, '');

		CREATE INDEX IF NOT EXISTS idx_payment_methods_method ON payment_methods(method);
		CREATE INDEX IF NOT EXISTS idx_payments_monitoring_date ON payments_monitoring(date);
		CREATE UNIQUE INDEX IF NOT EXISTS ux_payments_monitoring_order_method_amount_comment
			ON payments_monitoring(order_id, COALESCE(method, ''), amount, COALESCE(comment, ''))
			WHERE order_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_roles_name ON roles(name);
	`)
	if err != nil {
		log.Fatal("Failed to create indexes:", err)
	}
	log.Println("Indexes created successfully!")
}

func SeedTestData() {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	_, err := DB.Exec(
		"INSERT INTO users (username, password, role_id) VALUES ($1, $2, $3) ON CONFLICT (username) DO NOTHING",
		"admin", hashedPassword, 4,
	)
	if err != nil {
		log.Fatal("Failed to seed test user:", err)
	}

	paymentMethods := []string{
		"пч", "п ип", "ма", "аня", "над", "саша", "дима",
		"альфа", "втб", "сянь", "амин", "ачжу", "нал",
	}
	for _, method := range paymentMethods {
		_, err := DB.Exec("INSERT INTO payment_methods (method) VALUES ($1) ON CONFLICT (method) DO NOTHING", method)
		if err != nil {
			log.Fatal("Failed to seed payment method:", err)
		}
	}

	products := []models.Product{
		{
			ID:     6000,
			Status: 1,
			Name:   "Мешок № т5319. Ботинки. Швейцария. Арт 3630. Вес 27,7 кг. Кол-во 38 шт.",
			ArticlesInProduct: []models.ArticleInProduct{{ID: 0, Article: 3630, CursEvro: "99", PriceEvro: "18", Weight: "27.7", Count: 19, SumEvro: "498.6", SumRub: "49361"},
				{ID: 1, Article: 3246, CursEvro: "99", PriceEvro: "15.2", Weight: "27.7", Count: 19, SumEvro: "421.04", SumRub: "41653"}},
			Skidka:            "15",
			SummaRubSoSkidkoj: "41957",
			Count:             38,
			OnePrice:          "1104",
			Video:             "https://youtu.be/ccnjj9UjWZk",
			Description:       "Description",
		},
		{
			ID:                6001,
			Status:            1,
			Name:              "Мешок № х2928. Обувь микс Весна. Австрия. Арт 3246. Вес 31,9 кг. Кол-во 49 шт.",
			ArticlesInProduct: []models.ArticleInProduct{{ID: 0, Article: 3246, CursEvro: "99", PriceEvro: "15.2", Weight: "31.9", Count: 49, SumEvro: "433.2", SumRub: "42887"}},
			Skidka:            "30",
			SummaRubSoSkidkoj: "38402",
			Count:             49,
			OnePrice:          "784",
			Video:             "https://youtu.be/CS2E-pxUxoA",
			Description:       "Description2",
		},
	}

	for _, p := range products {
		_, err = DB.Exec(`
			INSERT INTO products (id, status, name, skidka, summaRubSoSkidkoj, count, oneprice, video, description)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO NOTHING
		`, p.ID, p.Status, p.Name, p.Skidka, p.SummaRubSoSkidkoj, p.Count, p.OnePrice, p.Video, p.Description)
		if err != nil {
			log.Fatal("Failed to insert product for seeding products:", err)
		}

		for _, article := range p.ArticlesInProduct {
			_, err := DB.Exec(`
				INSERT INTO article_in_product (product_id, article, cursEvro, priceEvro, weight, count, sumEvro, sumRub)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`, p.ID, article.Article, article.CursEvro, article.PriceEvro, article.Weight, article.Count, article.SumEvro, article.SumRub)
			if err != nil {
				log.Fatal("Failed to insert article for seeding products:", err)
			}
		}
	}

	_, err = DB.Exec(`
		INSERT INTO roles (name) VALUES ('ban'), ('worker'), ('manager'), ('admin')
		ON CONFLICT (name) DO NOTHING;

		INSERT INTO role_permissions (role_id, method, path, allowed)
		VALUES
		((SELECT id FROM roles WHERE name='worker'), 'DELETE', '/api/products/:id', false),
		((SELECT id FROM roles WHERE name='worker'), 'DELETE', '/api/orders/:id', false),
		((SELECT id FROM roles WHERE name='worker'), 'DELETE', '/api/payments/:id', false),
		((SELECT id FROM roles WHERE name='manager'), 'DELETE', '/api/payments_monitoring', false),
		((SELECT id FROM roles WHERE name='admin'), '*', '*', true)
		ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		log.Fatal("Failed to seed roles:", err)
	}
}

func LoginDB(storedUser *models.User) error {
	row := DB.QueryRow("SELECT id, username, password, role_id FROM users WHERE username = $1", storedUser.Username)
	err := row.Scan(&storedUser.ID, &storedUser.Username, &storedUser.Password, &storedUser.RoleID)
	if err != nil {
		return err
	}
	return nil
}

func GetUsersRoles() map[string]int {
	rows, err := DB.Query("SELECT username, role_id FROM users")
	if err != nil {
		log.Fatal("Failed to get users roles:", err)
	}
	defer rows.Close()

	usersRoles := make(map[string]int)
	for rows.Next() {
		var username string
		var roleID int
		if err := rows.Scan(&username, &roleID); err != nil {
			log.Fatal("Failed to scan user role:", err)
		}
		usersRoles[username] = roleID
	}
	return usersRoles
}

func GetUserRoleByID(userID int) (int, error) {
	var roleID int
	err := DB.QueryRow("SELECT role_id FROM users WHERE id = $1", userID).Scan(&roleID)
	if err != nil {
		return 0, err
	}
	return roleID, nil
}

func GetUserRoleByUsername(username string) (int, error) {
	var roleID int
	err := DB.QueryRow("SELECT role_id FROM users WHERE username = $1", username).Scan(&roleID)
	if err != nil {
		return 0, err
	}
	return roleID, nil
}

func GetRolesAllows() map[int]map[string]bool {
	rows, err := DB.Query("SELECT role_id, method, path, allowed FROM role_permissions")
	if err != nil {
		log.Fatal("Failed to get roles allows:", err)
	}
	defer rows.Close()

	rolesAllows := make(map[int]map[string]bool)
	for rows.Next() {
		var roleID int
		var method, path string
		var allowed bool
		if err := rows.Scan(&roleID, &method, &path, &allowed); err != nil {
			log.Fatal("Failed to scan role permission:", err)
		}
		if rolesAllows[roleID] == nil {
			rolesAllows[roleID] = make(map[string]bool)
		}
		rolesAllows[roleID][method+" "+path] = allowed
	}
	return rolesAllows
}
