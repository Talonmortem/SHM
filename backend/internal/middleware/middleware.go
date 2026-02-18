package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/Talonmortem/SHM/db"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

var JwtSecret = []byte("your-secret-key") // Replace with env variable in production

var UsersRoles map[string]int
var RolesAllows map[int]map[string]bool

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Username")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}
		c.Next()
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		username := c.GetHeader("X-Username")

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return JwtSecret, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if userIDRaw, exists := claims["user_id"]; exists {
				switch v := userIDRaw.(type) {
				case float64:
					c.Set("user_id", int(v))
				case int:
					c.Set("user_id", v)
				case string:
					parsed, parseErr := strconv.Atoi(v)
					if parseErr == nil {
						c.Set("user_id", parsed)
					}
				}
			}
			if tokenUsername, exists := claims["username"].(string); exists && tokenUsername != "" {
				username = tokenUsername
			}
			if tokenRoleIDRaw, exists := claims["role_id"]; exists {
				switch v := tokenRoleIDRaw.(type) {
				case float64:
					c.Set("role_id", int(v))
				case int:
					c.Set("role_id", v)
				case string:
					parsed, parseErr := strconv.Atoi(v)
					if parseErr == nil {
						c.Set("role_id", parsed)
					}
				}
			}
		}
		c.Set("username", username)
		c.Next()
	}
}

func RoleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.GetString("username")
		roleID := c.GetInt("role_id")

		if roleID == 0 {
			if userID, ok := c.Get("user_id"); ok {
				if id, ok := userID.(int); ok && id > 0 {
					liveRoleID, err := db.GetUserRoleByID(id)
					if err == nil {
						roleID = liveRoleID
					}
				}
			}
		}

		if roleID == 0 && username != "" {
			roleID = UsersRoles[username]
			if roleID == 0 {
				UsersRoles = db.GetUsersRoles()
				roleID = UsersRoles[username]
			}
			if roleID == 0 {
				liveRoleID, err := db.GetUserRoleByUsername(username)
				if err == nil {
					roleID = liveRoleID
					UsersRoles[username] = roleID
				}
			}
		}

		log.Printf("User: %s, RoleID: %d\n", username, roleID)

		if roleID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "role not found"})
			return
		}
		method := c.Request.Method
		path := c.FullPath()

		var allowed bool
		permissions, roleExists := RolesAllows[roleID]
		if !roleExists {
			RolesAllows = db.GetRolesAllows()
			permissions = RolesAllows[roleID]
		}

		// проверка с учётом wildcard '*'
		allowed, exists := permissions[method+" "+path]
		if !exists {
			allowed, exists = permissions[method+" *"]
		}
		if !exists {
			allowed, exists = permissions["* "+path]
		}
		if !exists {
			allowed, exists = permissions["* *"]
		}

		if !exists {
			// если явного разрешения нет → разрешаем по умолчанию
			allowed = true
		}

		if !allowed {
			log.Printf("Access denied: role=%d, method=%s, path=%s\n", roleID, method, path)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}

		c.Next()
	}
}

//Заполнить мапу ролей из базы

func LoadUsersRoles() {
	UsersRoles = db.GetUsersRoles()
	RolesAllows = db.GetRolesAllows()
	log.Println("Loaded user roles:", UsersRoles)
	log.Println("Loaded role permissions:", RolesAllows)
}
