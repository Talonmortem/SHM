package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/Talonmortem/SHM/db"
	"github.com/Talonmortem/SHM/internal/middleware"
	"github.com/Talonmortem/SHM/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var storedUser models.User
	storedUser.Username = user.Username
	err := db.LoginDB(&storedUser)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(storedUser.Password), []byte(user.Password)) != nil {
		log.Printf("Failed login attempt: %s with password: %s. stored: %s. error: %v", user.Username, user.Password, storedUser.Password, err)

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id":  storedUser.ID,
		"username": storedUser.Username,
		"role_id":  storedUser.RoleID,
		"exp":      time.Now().Add(time.Hour * 8).Unix(),
	})
	tokenString, err := token.SignedString(middleware.JwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString, "username": storedUser.Username, "role_id": storedUser.RoleID})
}

func LoginFormHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login",
	})
}
