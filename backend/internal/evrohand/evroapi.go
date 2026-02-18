package evrohand

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	cfg "github.com/Talonmortem/SHM/config"
)

type EvrohandApi struct {
	apiKey string
}

func NewEvrohandApi(cfg cfg.Config) *EvrohandApi {
	return &EvrohandApi{
		apiKey: cfg.APIKey,
	}
}

// ProductResponse структура для ответа API
type ProductResponse struct {
	Status string        `json:"status"`
	Data   []ProductData `json:"data"`
}

type ProductData struct {
	ID         int     `json:"id"`
	Added      string  `json:"added"`
	Lot        string  `json:"lot"`
	Name       string  `json:"name"`
	Art        string  `json:"art"`
	Cat        int     `json:"cat"`
	Categ      int     `json:"categ"`
	Weight     string  `json:"weight"`
	Count      string  `json:"count"`
	Preur      string  `json:"preur"`
	Kurs       string  `json:"kurs"`
	Prrub      string  `json:"prrub"`
	Prcosteur  string  `json:"prcosteur"`
	Prcostrub  string  `json:"prcostrub"`
	Totaleur   string  `json:"totaleur"`
	Totalrub   string  `json:"totalrub"`
	Skidka     string  `json:"skidka"`
	Totaleursk string  `json:"totaleursk"`
	Totalrubsk string  `json:"totalrubsk"`
	Link       string  `json:"link"`
	Status     int     `json:"status"`
	Tab        string  `json:"tab"`
	KursOwn    int     `json:"kurs_own"`
	PlLink     string  `json:"pl_link"`
	VideoPath  string  `json:"video_path"`
	Image      *string `json:"image"` // Может быть null
	CardLink   string  `json:"card_link"`
	Category   string  `json:"category"`
}

type Card struct {
	Lot         string
	Name        string
	Category    string
	Url         string
	Price       string
	EuroPrice   string
	RublePrice  string
	Weight      string
	PricePerKg  string
	CostPerItem string
	Count       string
	Status      string
	Skidka      string
	Youtubelink string
	Plink       string
}

/*
	"Предоставляю вам информацию о мешке, который вы упомянули -\n%s\nСсылка на лот: %s\n%s%s\nЦена €: %s\nЦена ₽: %s\nЦена за кг: %s\nСебестоимость одной вещи: %s\n%s\n%s\n",
	lotcard.Title, lotcard.Url, discountInfo, lotcard.Price, lotcard.EuroPrice, lotcard.RublePrice, lotcard.PricePerKg, lotcard.CostPerItem, lotcard.Youtubelink, lotcard.Button,
*/

// GetCardsInfo выполняет запрос к API и возвращает информацию о продукте.
func (api *EvrohandApi) GetCardsInfo(lot string) (cards []Card, err error) {
	url := fmt.Sprintf("https://evrohand.com/api/get_product.php?lot=%s&k=%s", lot, api.apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return []Card{}, fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []Card{}, fmt.Errorf("ошибка API: статус %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return []Card{}, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var product ProductResponse
	if err := json.Unmarshal(body, &product); err != nil {
		return []Card{}, fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if product.Status != "success" {
		return []Card{}, fmt.Errorf("ошибка API: статус %s", product.Status)
	}

	//log.Printf("API response: %v", product)

	for _, product := range product.Data {
		cards = append(cards, extractProductToCard(product))
	}

	return cards, nil
}

func extractProductToCard(product ProductData) Card {

	var status string
	var skidka int
	var skidkastr string

	switch product.Status {
	case 1:
		status = "В наличии"
	case 2:
		status = "Забронировано"
	case 3:
		status = "Продано"
	}

	if product.Skidka != "" {
		skidka, _ = strconv.Atoi(product.Skidka)
		skidkastr = "Скидка " + product.Skidka + " %"
	}

	//weight*preur - price eur
	//weight*preur*kurs - price rub
	preur, err := strconv.ParseFloat(product.Preur, 64)
	if err != nil {
		fmt.Printf("Ошибка парсинга preur: %v\n", err)
	}
	weight, err := strconv.ParseFloat(product.Weight, 64)
	if err != nil {
		fmt.Printf("Ошибка парсинга weight: %v\n", err)
	}
	kurs, err := strconv.ParseFloat(product.Kurs, 64)
	if err != nil {
		fmt.Printf("Ошибка парсинга weight: %v\n", err)
	}

	totaleur := (weight * preur)
	totalrub := totaleur * kurs
	totalrubprice := (totalrub / 100.00) * (100 - float64(skidka))
	//totaleurprice := (totaleur / 100.00) * (100 - float64(skidka))

	var totalrubpricestr, totaleurpricestr string

	if skidka > 0 {
		//totaleurpricestr = "Цена: " + fmt.Sprintf("%.0f", totaleur) + " €" + "\n" + "Цена со скидкой: " + fmt.Sprintf("%.0f", totaleurprice) + " €"
		totalrubpricestr = "Цена ₽: " + fmt.Sprintf("%.2f", totalrub) + "" + "\n" + "Цена со скидкой ₽: " + fmt.Sprintf("%.0f", totalrubprice) + " "
		//priceperkgstr = "Цена за кг: " + fmt.Sprintf("%.0f", preur*kurs) + " ₽" + "\n" + "Цена за кг со скидкой: " + fmt.Sprintf("%.0f", (preur*kurs/100)*(100-float64(skidka))) + " ₽"
	} else {
		totalrubpricestr = "Цена ₽: " + fmt.Sprintf("%.2f", totalrubprice)
		//priceperkgstr = "Цена за кг: " + fmt.Sprintf("%.0f", preur) + " €"
	}
	priceperkgstr := "Цена за кг €: " + fmt.Sprintf("%.2f", preur) + " "
	totaleurpricestr = "Цена €: " + fmt.Sprintf("%.2f", totaleur) + " €"

	return Card{
		Lot:         product.Lot,
		Name:        product.Name,
		Category:    product.Category,
		Url:         product.CardLink,
		EuroPrice:   totaleurpricestr,
		RublePrice:  totalrubpricestr,
		PricePerKg:  priceperkgstr,
		Weight:      product.Weight + " кг",
		Count:       product.Count,
		Status:      status,
		Skidka:      skidkastr,
		Youtubelink: product.Link,
		Plink:       product.PlLink,
	}
}

func (api *EvrohandApi) IsExistLotNumber(lot string) bool {
	cards, err := api.GetCardsInfo(lot)
	if err != nil {
		log.Printf("Error genegate name: %v", err)
	}
	return len(cards) > 0
}

func (api *EvrohandApi) FindLot(name string, wordmap *map[string]Card) error {
	url := fmt.Sprintf("https://evrohand.com/api/get_product.php?name=%s&k=%s", name, api.apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("ошибка при выполнении запроса: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ошибка API: статус %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	var product ProductResponse
	if err := json.Unmarshal(body, &product); err != nil {
		return fmt.Errorf("ошибка парсинга JSON: %v", err)
	}

	if product.Status != "success" {
		return fmt.Errorf("ошибка API: статус %s", product.Status)
	}

	for _, product := range product.Data {
		(*wordmap)[name] = extractProductToCard(product)
	}

	return nil
}
