package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
	RoleID   int    `json:"role_id"`
}

type Product struct {
	ID                int                `json:"id"`
	Status            int                `json:"status"`
	Name              string             `json:"name"`
	ArticlesInProduct []ArticleInProduct `json:"articlesInProduct"`
	Skidka            string             `json:"skidka"`
	SummaRubSoSkidkoj string             `json:"summaRubSoSkidkoj"`
	Count             int                `json:"count"`
	OnePrice          string             `json:"onePrice"`
	Video             string             `json:"video"`
	Description       string             `json:"description"`
}

type ArticleInProduct struct {
	ID        int    `json:"id"`
	Article   int    `json:"article"`
	CursEvro  string `json:"cursEvro"`
	PriceEvro string `json:"priceEvro"`
	Weight    string `json:"weight"`
	Count     int    `json:"count"`
	SumEvro   string `json:"sumEvro"` //price * weight
	SumRub    string `json:"sumRub"`
}

type Article struct {
	ID          int     `json:"id"`
	Code        int     `json:"code"`
	Description string  `json:"description"`
	Euro        float64 `json:"euro"`
	Count       int     `json:"count"`
	Weight      float64 `json:"weight"`
	Price       float64 `json:"price"`
}

func (a *Article) UnmarshalJSON(data []byte) error {
	type rawArticle struct {
		ID          interface{} `json:"id"`
		Code        interface{} `json:"code"`
		Description string      `json:"description"`
		Euro        interface{} `json:"euro"`
		Count       interface{} `json:"count"`
		Weight      interface{} `json:"weight"`
		Price       interface{} `json:"price"`
	}

	var raw rawArticle
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	a.Description = raw.Description

	if raw.ID != nil {
		id, err := parseFlexibleInt(raw.ID)
		if err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
		a.ID = id
	}
	if raw.Code != nil {
		code, err := parseFlexibleInt(raw.Code)
		if err != nil {
			return fmt.Errorf("invalid code: %w", err)
		}
		a.Code = code
	}
	if raw.Euro != nil {
		euro, err := parseFlexibleFloat(raw.Euro)
		if err != nil {
			return fmt.Errorf("invalid euro: %w", err)
		}
		a.Euro = euro
	}
	if raw.Count != nil {
		count, err := parseFlexibleInt(raw.Count)
		if err != nil {
			return fmt.Errorf("invalid count: %w", err)
		}
		a.Count = count
	}
	if raw.Weight != nil {
		weight, err := parseFlexibleFloat(raw.Weight)
		if err != nil {
			return fmt.Errorf("invalid weight: %w", err)
		}
		a.Weight = weight
	}
	if raw.Price != nil {
		price, err := parseFlexibleFloat(raw.Price)
		if err != nil {
			return fmt.Errorf("invalid price: %w", err)
		}
		a.Price = price
	}

	return nil
}

func parseFlexibleInt(v interface{}) (int, error) {
	switch value := v.(type) {
	case float64:
		return int(value), nil
	case int:
		return value, nil
	case string:
		if strings.TrimSpace(value) == "" {
			return 0, nil
		}
		normalized := normalizeNumericString(value)
		if normalized == "" {
			return 0, fmt.Errorf("empty numeric value")
		}
		n, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			return 0, err
		}
		return int(n), nil
	default:
		return 0, fmt.Errorf("unsupported value type %T", v)
	}
}

func parseFlexibleFloat(v interface{}) (float64, error) {
	switch value := v.(type) {
	case float64:
		return value, nil
	case int:
		return float64(value), nil
	case string:
		if strings.TrimSpace(value) == "" {
			return 0, nil
		}
		normalized := normalizeNumericString(value)
		if normalized == "" {
			return 0, fmt.Errorf("empty numeric value")
		}
		n, err := strconv.ParseFloat(normalized, 64)
		if err != nil {
			return 0, err
		}
		return n, nil
	default:
		return 0, fmt.Errorf("unsupported value type %T", v)
	}
}

func normalizeNumericString(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	var cleaned strings.Builder
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == ',' || r == '.' || r == '-' || r == '+' {
			cleaned.WriteRune(r)
		}
	}
	s = cleaned.String()
	if s == "" {
		return ""
	}

	lastComma := strings.LastIndex(s, ",")
	lastDot := strings.LastIndex(s, ".")

	switch {
	case lastComma >= 0 && lastDot >= 0:
		if lastComma > lastDot {
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			s = strings.ReplaceAll(s, ",", "")
		}
	case lastComma >= 0:
		s = strings.ReplaceAll(s, ",", ".")
	}

	if strings.Count(s, ".") > 1 {
		last := strings.LastIndex(s, ".")
		intPart := strings.ReplaceAll(s[:last], ".", "")
		fracPart := strings.ReplaceAll(s[last+1:], ".", "")
		s = intPart + "." + fracPart
	}

	return s
}

type Payment struct { // corresponds to payments_monitoring table
	ID      int     `json:"id"`
	Date    string  `json:"date"`
	Method  string  `json:"method"`
	OrderID int     `json:"order_id"`
	Amount  float64 `json:"amount"`
	Comment string  `json:"comment"`
}

type Order struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Components  []Product `json:"components"` // JSON array of component objects
	Quantity    int       `json:"quantity"`
	Status      int       `json:"status"`
	Description string    `json:"description"`
	Payments    []Payment `json:"payments"` // JSON array of payment objects as string
	Debt        float64   `json:"debt"`
}

type Client struct {
	ID             int    `json:"id"`
	City           string `json:"city"`
	FullName       string `json:"full_name"`
	Phone          string `json:"phone"`
	PassportNumber string `json:"passport_number"`
	TK             string `json:"tk"`
	Comment        string `json:"comment"`
}

type Shipment struct {
	ID          int     `json:"id"`
	ShipDate    string  `json:"ship_date"`
	City        string  `json:"city"`
	FullName    string  `json:"full_name"`
	Phone       string  `json:"phone"`
	PassportInn string  `json:"passport_inn"`
	TK          string  `json:"tk"`
	Places      int     `json:"places"`
	Price       float64 `json:"price"`
	Weight      float64 `json:"weight"`
}

type ShipmentNote struct {
	ID       int    `json:"id"`
	ShipDate string `json:"ship_date"`
	Note     string `json:"note"`
}
