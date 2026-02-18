package db

import (
	"fmt"
	"strings"
	"time"
)

const paymentDateTimeLayout = "2006-01-02 15:04:05"

func currentPaymentDateTime() string {
	return time.Now().Format(paymentDateTimeLayout)
}

func normalizePaymentDateInput(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return currentPaymentDateTime(), nil
	}

	layouts := []string{
		paymentDateTimeLayout,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
		"02-01-2006 15:04:05",
		"02-01-2006",
	}

	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Format(paymentDateTimeLayout), nil
		}
	}

	return "", fmt.Errorf("unsupported payment date format: %q", raw)
}
