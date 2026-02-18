package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/Talonmortem/SHM/db"
	"github.com/Talonmortem/SHM/internal/models"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Получаем имя пользователя из контекста (AuthMiddleware должен положить его)
		username, _ := c.Get("username") // если нет, будет nil

		// 2. Собираем параметры
		params := make(map[string]interface{})

		// Query параметры
		for k, v := range c.Request.URL.Query() {
			if len(v) == 1 {
				params[k] = v[0]
			} else {
				params[k] = v
			}
		}

		// Body параметры (если есть JSON)
		if c.Request.Body != nil {
			// Читаем тело запроса, чтобы не потерять
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			if len(bodyBytes) > 0 {
				var bodyData interface{}
				if err := json.Unmarshal(bodyBytes, &bodyData); err == nil {
					params["body"] = bodyData
				}
				// Восстанавливаем Body для последующих хэндлеров
				c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}
		}

		// 3. Сохраняем лог асинхронно
		go func() {
			paramsJSON, _ := json.Marshal(params)

			logEntry := models.RequestLog{
				Username:  username.(string),
				Method:    c.Request.Method,
				Path:      c.FullPath(),
				Params:    paramsJSON,
				CreatedAt: time.Now(),
			}

			// сохраняем через db-пакет
			if err := db.SaveRequestLog(&logEntry); err != nil {
				// тут можно сделать fallback в файл, если надо
			}
		}()

		c.Next()
	}
}
