package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/Talonmortem/SHM/config"
	"github.com/Talonmortem/SHM/internal/evrohand"
	"github.com/Talonmortem/SHM/internal/utils"
	"github.com/gin-gonic/gin"
)

var errProductNameNotGenerated = errors.New("could not generate unique product name")

func generateUniqueProductName(ctx context.Context) (string, error) {
	const maxAttempts = 100
	api := evrohand.NewEvrohandApi(config.Load())

	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}

		candidate := utils.GenerateRandomString()
		if !api.IsExistLotNumber(candidate) {
			return candidate, nil
		}
	}

	return "", errProductNameNotGenerated
}

func GenerateProductNameHandler(c *gin.Context) {
	name, err := generateUniqueProductName(c.Request.Context())
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "request canceled"})
			return
		}
		if errors.Is(err, errProductNameNotGenerated) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"name": name})
}
