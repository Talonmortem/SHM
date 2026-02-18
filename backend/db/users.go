package db

import (
	"log"

	"github.com/Talonmortem/SHM/internal/models"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func CreateTablesUsers() {
	tx, err := DB.Begin()
	if err != nil {
		log.Fatal("Failed to begin transaction:", err)
	}

	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			username TEXT UNIQUE,
			password TEXT,
			role_id INTEGER
		);

		CREATE TABLE IF NOT EXISTS roles (
			id BIGSERIAL PRIMARY KEY,
			name TEXT UNIQUE
		);

		CREATE TABLE IF NOT EXISTS role_permissions (
			id BIGSERIAL PRIMARY KEY,
			role_id BIGINT NOT NULL REFERENCES roles(id),
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			allowed BOOLEAN NOT NULL DEFAULT true,
			UNIQUE (role_id, method, path)
		);
	`)
	if err != nil {
		tx.Rollback()
		log.Fatal("Failed to create tables:", err)
	}
	tx.Commit()
	log.Println("Tables users created successfully!")
}

func CreateUser(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = string(hashedPassword)

	err = DB.QueryRow(
		"INSERT INTO users (username, password, role_id) VALUES ($1, $2, $3) RETURNING id",
		user.Username, user.Password, user.RoleID,
	).Scan(&user.ID)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(201, user)
}

func GetUsers(c *gin.Context) {
	rows, err := DB.Query("SELECT id, username, role_id FROM users")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.ID, &user.Username, &user.RoleID); err != nil {
			c.JSON(500, gin.H{"error": "Failed to scan user"})
			return
		}
		users = append(users, user)
	}
	c.JSON(200, users)
}

func UpdateUser(c *gin.Context) {
	id := c.Param("id")
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if user.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to hash password"})
			return
		}
		user.Password = string(hashedPassword)
		_, err = DB.Exec("UPDATE users SET username = $1, password = $2, role_id = $3 WHERE id = $4", user.Username, user.Password, user.RoleID, id)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to update user"})
			return
		}
	} else {
		_, err := DB.Exec("UPDATE users SET username = $1, role_id = $2 WHERE id = $3", user.Username, user.RoleID, id)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to update user"})
			return
		}
	}

	c.JSON(200, gin.H{"message": "User updated successfully"})
}

func GetRoles(c *gin.Context) {
	rows, err := DB.Query("SELECT id, name FROM roles")
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch roles"})
		return
	}
	defer rows.Close()

	var roles []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	for rows.Next() {
		var role struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			c.JSON(500, gin.H{"error": "Failed to scan role"})
			return
		}
		roles = append(roles, role)
	}
	c.JSON(200, roles)
}
