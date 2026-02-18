package utils

import (
	"fmt"
	"math/rand"
)

// Генерация случайного значения в формате [буква]+4 цифры
func GenerateRandomString() string {
	letters := []rune("абвгдежзийклмнопрстуфхцчшщэюя")
	letter := letters[rand.Intn(len(letters))]
	number := rand.Intn(10000) // диапазон от 0000 до 9999

	//fmt.Printf("Сгенерированное значение: %c%04d\n", letter, number)
	// Форматируем в виде [буква]+[4 цифры]
	return fmt.Sprintf("%c%04d", letter, number)
}
