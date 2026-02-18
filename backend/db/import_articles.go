package db

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type ImportArticlesResult struct {
	Inserted int
	Skipped  int
}

func ImportArticlesFromCSV(path string) (ImportArticlesResult, error) {
	file, err := os.Open(path)
	if err != nil {
		return ImportArticlesResult{}, fmt.Errorf("open csv file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	tx, err := DB.Begin()
	if err != nil {
		return ImportArticlesResult{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO articles (id, no, code, description, euro, colli, kg, value)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`)
	if err != nil {
		return ImportArticlesResult{}, fmt.Errorf("prepare insert statement: %w", err)
	}
	defer stmt.Close()

	result := ImportArticlesResult{}
	lineNo := 0
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return result, fmt.Errorf("read csv line %d: %w", lineNo+1, err)
		}
		lineNo++

		article, ok := parseCSVArticleRecord(record)
		if !ok {
			result.Skipped++
			continue
		}

		if _, err := stmt.Exec(article.ID, article.No, article.Code, article.Description, article.Euro, article.Colli, article.KG, article.Value); err != nil {
			return result, fmt.Errorf("insert csv line %d: %w", lineNo, err)
		}

		result.Inserted++
	}

	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("commit transaction: %w", err)
	}

	return result, nil
}

func TruncateArticles() error {
	_, err := DB.Exec("TRUNCATE TABLE articles RESTART IDENTITY CASCADE")
	if err != nil {
		return fmt.Errorf("truncate articles: %w", err)
	}
	return nil
}

func parseCSVArticleRecord(record []string) (ArticleCSVImportRow, bool) {
	if len(record) < 8 {
		return ArticleCSVImportRow{}, false
	}

	idRaw := strings.TrimSpace(record[0])
	if idRaw == "" {
		return ArticleCSVImportRow{}, false
	}

	id, err := parseCSVInt(idRaw)
	if err != nil || id <= 0 {
		log.Printf("skip line with invalid ID %q: %v", idRaw, err)
		return ArticleCSVImportRow{}, false
	}

	no, err := parseCSVInt(strings.TrimSpace(record[1]))
	if err != nil {
		no = 0
	}

	code := strings.TrimSpace(record[2])
	description := strings.TrimSpace(record[3])

	euro, err := parseCSVFloat(strings.TrimSpace(record[4]))
	if err != nil {
		log.Printf("skip ID %d due to invalid EURO %q: %v", id, record[4], err)
		return ArticleCSVImportRow{}, false
	}
	colli, err := parseCSVFloat(strings.TrimSpace(record[5]))
	if err != nil {
		log.Printf("skip ID %d due to invalid COLLI %q: %v", id, record[5], err)
		return ArticleCSVImportRow{}, false
	}
	kg, err := parseCSVFloat(strings.TrimSpace(record[6]))
	if err != nil {
		log.Printf("skip ID %d due to invalid KG %q: %v", id, record[6], err)
		return ArticleCSVImportRow{}, false
	}
	value, err := parseCSVFloat(strings.TrimSpace(record[7]))
	if err != nil {
		log.Printf("skip ID %d due to invalid VALUE %q: %v", id, record[7], err)
		return ArticleCSVImportRow{}, false
	}

	return ArticleCSVImportRow{
		ID:          id,
		No:          no,
		Code:        code,
		Description: description,
		Euro:        euro,
		Colli:       colli,
		KG:          kg,
		Value:       value,
	}, true
}

type ArticleCSVImportRow struct {
	ID          int
	No          int
	Code        string
	Description string
	Euro        float64
	Colli       float64
	KG          float64
	Value       float64
}

func parseCSVInt(raw string) (int, error) {
	normalized := normalizeCSVNumber(raw)
	if normalized == "" {
		return 0, nil
	}
	f, err := strconv.ParseFloat(normalized, 64)
	if err != nil {
		return 0, err
	}
	return int(f), nil
}

func parseCSVFloat(raw string) (float64, error) {
	normalized := normalizeCSVNumber(raw)
	if normalized == "" {
		return 0, nil
	}
	return strconv.ParseFloat(normalized, 64)
}

func normalizeCSVNumber(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	var b strings.Builder
	for _, r := range raw {
		switch {
		case (r >= '0' && r <= '9') || r == ',' || r == '.' || r == '-' || r == '+':
			b.WriteRune(r)
		case unicode.IsSpace(r):
			continue
		default:
			continue
		}
	}

	normalized := strings.ReplaceAll(b.String(), ",", ".")
	if strings.Count(normalized, ".") > 1 {
		parts := strings.Split(normalized, ".")
		normalized = strings.Join(parts[:len(parts)-1], "") + "." + parts[len(parts)-1]
	}

	return normalized
}
