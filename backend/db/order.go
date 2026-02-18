package db

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Talonmortem/SHM/internal/models"
	"github.com/gin-gonic/gin"
)

func GetOrders(c *gin.Context) {
	query := `
        SELECT o.id, o.name, o.quantity, o.status, o.description, o.debt,
               o.ship_date, o.city, o.full_name, o.phone, o.passport_inn, o.tk, o.places, o.price, o.weight,
               p.id, p.status, p.name, p.video, p.weight, p.skidka, p.summaRubSoSkidkoj, p.count, p.onePrice, p.description,
               pm.id, pm.date, pm.method, pm.amount, pm.comment
        FROM orders o
        LEFT JOIN order_products op ON o.id = op.order_id
        LEFT JOIN products p ON op.product_id = p.id
        LEFT JOIN payments_monitoring pm ON o.id = pm.order_id
    `
	rows, err := DB.Query(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	ordersMap := make(map[int]*models.Order)
	componentsMaps := make(map[int]map[int]models.Product)
	paymentsMap := make(map[int]map[int]models.Payment)

	for rows.Next() {
		var o models.Order
		var p models.Product
		var pm models.Payment
		var debt sql.NullFloat64
		var shipDate, city, fullName, phone, passportInn, tk sql.NullString
		var places sql.NullInt64
		var price, weight sql.NullFloat64
		var productID, paymentID sql.NullInt64
		var productStatus, productCount sql.NullInt64
		var productName, productVideo, productWeight, productSkidka, productSummaRubSoSkidkoj, productOnePrice, productDescription sql.NullString
		var paymentMethod, paymentDate, paymentComment sql.NullString
		var paymentAmount sql.NullFloat64
		err := rows.Scan(
			&o.ID, &o.Name, &o.Quantity, &o.Status, &o.Description, &debt,
			&shipDate, &city, &fullName, &phone, &passportInn, &tk, &places, &price, &weight,
			&productID, &productStatus, &productName, &productVideo, &productWeight, &productSkidka, &productSummaRubSoSkidkoj, &productCount, &productOnePrice, &productDescription,
			&paymentID, &paymentDate, &paymentMethod, &paymentAmount, &paymentComment,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if _, exists := ordersMap[o.ID]; !exists {
			if debt.Valid {
				o.Debt = debt.Float64
			}
			if shipDate.Valid {
				o.ShipDate = shipDate.String
			}
			if city.Valid {
				o.City = city.String
			}
			if fullName.Valid {
				o.FullName = fullName.String
			}
			if phone.Valid {
				o.Phone = phone.String
			}
			if passportInn.Valid {
				o.PassportInn = passportInn.String
			}
			if tk.Valid {
				o.TK = tk.String
			}
			if places.Valid {
				o.Places = int(places.Int64)
			}
			if price.Valid {
				o.Price = price.Float64
			}
			if weight.Valid {
				o.Weight = weight.Float64
			}
			ordersMap[o.ID] = &o
			componentsMaps[o.ID] = make(map[int]models.Product)
			paymentsMap[o.ID] = make(map[int]models.Payment)
		}

		if productID.Valid && productName.Valid {
			p.ID = int(productID.Int64)
			p.Name = productName.String
			if productStatus.Valid {
				p.Status = int(productStatus.Int64)
			}
			if productVideo.Valid {
				p.Video = productVideo.String
			}
			if productWeight.Valid {
				p.Weight = productWeight.String
			}
			if productSkidka.Valid {
				p.Skidka = productSkidka.String
			}
			if productSummaRubSoSkidkoj.Valid {
				p.SummaRubSoSkidkoj = productSummaRubSoSkidkoj.String
			}
			if productCount.Valid {
				p.Count = int(productCount.Int64)
			}
			if productOnePrice.Valid {
				p.OnePrice = productOnePrice.String
			}
			if productDescription.Valid {
				p.Description = productDescription.String
			}
			componentsMaps[o.ID][p.ID] = p
		}

		if paymentID.Valid && paymentMethod.Valid {
			pm.ID = int(paymentID.Int64)
			pm.Method = paymentMethod.String
			if paymentDate.Valid {
				pm.Date = paymentDate.String
			}
			if paymentAmount.Valid {
				pm.Amount = paymentAmount.Float64
			}
			if paymentComment.Valid {
				pm.Comment = paymentComment.String
			}
			paymentsMap[o.ID][pm.ID] = pm
		}
	}

	var orders []models.Order
	for id, o := range ordersMap {
		for _, p := range componentsMaps[id] {
			articleRows, err := DB.Query("SELECT id, article, cursEvro, priceEvro, weight, count, sumEvro, sumRub FROM article_in_product WHERE product_id = $1", p.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			for articleRows.Next() {
				var article models.ArticleInProduct
				if err := articleRows.Scan(&article.ID, &article.Article, &article.CursEvro, &article.PriceEvro, &article.Weight, &article.Count, &article.SumEvro, &article.SumRub); err != nil {
					articleRows.Close()
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				p.ArticlesInProduct = append(p.ArticlesInProduct, article)
			}
			articleRows.Close()
			o.Components = append(o.Components, p)
		}
		for _, pm := range paymentsMap[id] {
			o.Payments = append(o.Payments, pm)
		}
		o.Debt = countDebt(o.Components, o.Payments)
		orders = append(orders, *o)
	}

	c.JSON(http.StatusOK, orders)
}

func countDebt(component []models.Product, payments []models.Payment) float64 {
	totalOrderAmount := 0.0
	for _, product := range component {
		price, err := parseAmount(product.SummaRubSoSkidkoj)
		if err != nil {
			log.Println("Failed to parse product price:", err)
			continue
		}
		totalOrderAmount += price
	}

	totalPaid := 0.0
	for _, payment := range payments {
		totalPaid += payment.Amount
	}

	debt := totalOrderAmount - totalPaid
	return debt
}

func totalPaid(payments []models.Payment) float64 {
	total := 0.0
	for _, payment := range payments {
		total += payment.Amount
	}
	return total
}

func countOrderAmountByProductIDs(tx *sql.Tx, productIDs []int) (float64, error) {
	total := 0.0
	for _, productID := range productIDs {
		var raw sql.NullString
		if err := tx.QueryRow("SELECT summaRubSoSkidkoj FROM products WHERE id = $1", productID).Scan(&raw); err != nil {
			return 0, err
		}
		value := ""
		if raw.Valid {
			value = raw.String
		}
		price, err := parseAmount(value)
		if err != nil {
			log.Printf("Failed to parse product price for product %d: %v", productID, err)
			continue
		}
		total += price
	}
	return total, nil
}

func recalculateOrderDebt(tx *sql.Tx, orderID int) error {
	rows, err := tx.Query("SELECT product_id FROM order_products WHERE order_id = $1", orderID)
	if err != nil {
		return err
	}
	defer rows.Close()

	productIDs := make([]int, 0)
	for rows.Next() {
		var productID int
		if err := rows.Scan(&productID); err != nil {
			return err
		}
		productIDs = append(productIDs, productID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	totalOrderAmount, err := countOrderAmountByProductIDs(tx, productIDs)
	if err != nil {
		return err
	}

	var totalPaid sql.NullFloat64
	if err := tx.QueryRow("SELECT COALESCE(SUM(amount), 0) FROM payments_monitoring WHERE order_id = $1", orderID).Scan(&totalPaid); err != nil {
		return err
	}

	debt := totalOrderAmount
	if totalPaid.Valid {
		debt -= totalPaid.Float64
	}

	_, err = tx.Exec("UPDATE orders SET quantity = $1, debt = $2 WHERE id = $3", len(productIDs), debt, orderID)
	return err
}

func parseAmount(raw string) (float64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, nil
	}

	// Support numbers with spaces and comma decimal separators from UI.
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, ",", ".")

	return strconv.ParseFloat(value, 64)
}

func collectUniqueProductIDs(components []models.Product) ([]int, error) {
	if len(components) == 0 {
		return nil, fmt.Errorf("at least one product must be selected")
	}

	seen := make(map[int]struct{}, len(components))
	productIDs := make([]int, 0, len(components))
	for _, product := range components {
		if product.ID <= 0 {
			return nil, fmt.Errorf("invalid product ID: %d", product.ID)
		}
		if _, exists := seen[product.ID]; exists {
			return nil, fmt.Errorf("duplicate product in order: %d", product.ID)
		}
		seen[product.ID] = struct{}{}
		productIDs = append(productIDs, product.ID)
	}
	return productIDs, nil
}

func validateOrderProductSelection(tx *sql.Tx, orderID int, productIDs []int, existingOrderProducts map[int]struct{}) error {
	for _, productID := range productIDs {
		var status int
		if err := tx.QueryRow("SELECT status FROM products WHERE id = $1", productID).Scan(&status); err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("product %d does not exist", productID)
			}
			return err
		}

		_, alreadyInOrder := existingOrderProducts[productID]
		if alreadyInOrder {
			if status == 3 {
				return fmt.Errorf("product %d is already sold and cannot remain in the order", productID)
			}
			continue
		}

		if status != 1 {
			return fmt.Errorf("product %d is not available for adding to the order", productID)
		}

		var linkedOrderID int
		err := tx.QueryRow("SELECT order_id FROM order_products WHERE product_id = $1 LIMIT 1", productID).Scan(&linkedOrderID)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		if err == nil && linkedOrderID != orderID {
			return fmt.Errorf("product %d is already linked to another order", productID)
		}
	}

	return nil
}

func productStatusForOrder(orderStatus int) int {
	if orderStatus == 2 {
		return 3
	}
	return 2
}

func CreateOrder(c *gin.Context) {
	var order models.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if order.Status < 0 || order.Status > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be 0 (Новый), 1 (Готов к отправке), or 2 (Отправлен)"})
		return
	}

	for _, p := range order.Payments {
		var exists bool
		err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM payment_methods WHERE method = $1)", p.Method).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid partial payment method: " + p.Method})
			return
		}
		if p.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Partial payment amount must be positive"})
			return
		}
	}

	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin transaction: " + err.Error()})
		return
	}
	defer tx.Rollback()

	productIDs, err := collectUniqueProductIDs(order.Components)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateOrderProductSelection(tx, 0, productIDs, map[int]struct{}{}); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var orderID int
	err = tx.QueryRow(`
		INSERT INTO orders (
			name, quantity, status, description, debt, ship_date, city, full_name, phone, passport_inn, tk, places, price, weight
		)
		VALUES ($1, 0, $2, $3, 0, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`, order.Name, order.Status, order.Description, order.ShipDate, order.City, order.FullName, order.Phone, order.PassportInn, order.TK, order.Places, order.Price, order.Weight).Scan(&orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order: " + err.Error()})
		return
	}

	log.Printf("\nOrder components: %v\n", order.Components)

	targetProductStatus := productStatusForOrder(order.Status)
	for _, product := range order.Components {
		_, err := tx.Exec("UPDATE products SET status = $1 WHERE id = $2", targetProductStatus, product.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product status: " + err.Error()})
			return
		}

		_, err = tx.Exec("INSERT INTO order_products (order_id, product_id) VALUES ($1, $2)", orderID, product.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link product to order: " + err.Error()})
			return
		}
		log.Printf("Товар с ID %d добавлен в заказ с ID %d\n", product.ID, orderID)
	}

	for i := range order.Payments {
		p := &order.Payments[i]
		paymentTimestamp := currentPaymentDateTime()
		err := tx.QueryRow(
			"INSERT INTO payments_monitoring (date, method, order_id, amount, comment) VALUES ($1, $2, $3, $4, $5) RETURNING id",
			paymentTimestamp, p.Method, orderID, p.Amount, p.Comment,
		).Scan(&p.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert payment: " + err.Error()})
			return
		}
		p.Date = paymentTimestamp
		log.Printf("Платеж с ID %d с параметрами %v создан\n", p.ID, p)
	}

	order.Quantity = len(order.Components)
	totalOrderAmount, err := countOrderAmountByProductIDs(tx, productIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate order amount: " + err.Error()})
		return
	}
	order.Debt = totalOrderAmount - totalPaid(order.Payments)

	_, err = tx.Exec("UPDATE orders SET quantity = $1, debt = $2 WHERE id = $3", order.Quantity, order.Debt, orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order: " + err.Error()})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	order.ID = int(orderID)
	log.Printf("Order created with ID: %d\n", order.ID)
	c.JSON(http.StatusOK, order)
}

func UpdateOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}
	var order models.Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if order.Status < 0 || order.Status > 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status must be 0 (Новый), 1 (Готов к отправке), or 2 (Отправлен)"})
		return
	}

	for _, p := range order.Payments {
		var exists bool
		err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM payment_methods WHERE method = $1)", p.Method).Scan(&exists)
		if err != nil || !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid partial payment method: " + p.Method})
			return
		}
		if p.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Partial payment amount must be positive"})
			return
		}
	}

	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction: " + err.Error()})
		return
	}
	defer tx.Rollback()

	var oldOrderStatus int
	if err := tx.QueryRow("SELECT status FROM orders WHERE id = $1", id).Scan(&oldOrderStatus); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch old order status: " + err.Error()})
		return
	}

	var oldProductIDs []int
	oldRows, err := tx.Query("SELECT product_id FROM order_products WHERE order_id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch old components: " + err.Error()})
		return
	}
	for oldRows.Next() {
		var pid int
		if err := oldRows.Scan(&pid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		oldProductIDs = append(oldProductIDs, pid)
	}
	oldRows.Close()

	existingOrderProducts := make(map[int]struct{}, len(oldProductIDs))
	for _, pid := range oldProductIDs {
		existingOrderProducts[pid] = struct{}{}
	}

	productIDs, err := collectUniqueProductIDs(order.Components)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validateOrderProductSelection(tx, id, productIDs, existingOrderProducts); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newProductIDs := make(map[int]bool)
	for _, p := range order.Components {
		newProductIDs[p.ID] = true
	}

	for _, pid := range oldProductIDs {
		if !newProductIDs[pid] {
			_, err := tx.Exec("UPDATE products SET status = 1 WHERE id = $1", pid)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset product status: " + err.Error()})
				return
			}
		}
	}

	targetProductStatus := productStatusForOrder(order.Status)
	for _, p := range order.Components {
		isNew := true
		for _, opid := range oldProductIDs {
			if opid == p.ID {
				isNew = false
				break
			}
		}
		if isNew {
			_, err := tx.Exec("UPDATE products SET status = $1 WHERE id = $2", targetProductStatus, p.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product status: " + err.Error()})
				return
			}
		}
	}

	if oldOrderStatus != 2 && order.Status == 2 {
		for _, p := range order.Components {
			_, err := tx.Exec("UPDATE products SET status = 3 WHERE id = $1", p.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark product as sold: " + err.Error()})
				return
			}
		}
	}

	_, err = tx.Exec("DELETE FROM order_products WHERE order_id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete old components: " + err.Error()})
		return
	}

	for _, p := range order.Components {
		_, err = tx.Exec("INSERT INTO order_products (order_id, product_id) VALUES ($1, $2)", id, p.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to associate product with order: " + err.Error()})
			return
		}
	}

	existingPaymentIDs := map[int]struct{}{}
	existingPaymentsRows, err := tx.Query("SELECT id FROM payments_monitoring WHERE order_id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch existing payments: " + err.Error()})
		return
	}
	for existingPaymentsRows.Next() {
		var paymentID int
		if err := existingPaymentsRows.Scan(&paymentID); err != nil {
			existingPaymentsRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse existing payments: " + err.Error()})
			return
		}
		existingPaymentIDs[paymentID] = struct{}{}
	}
	existingPaymentsRows.Close()

	keptPaymentIDs := map[int]struct{}{}
	for i := range order.Payments {
		p := &order.Payments[i]
		paymentTimestamp := currentPaymentDateTime()
		if p.ID > 0 {
			res, err := tx.Exec("UPDATE payments_monitoring SET date = $1, method = $2, amount = $3, comment = $4 WHERE id = $5 AND order_id = $6",
				paymentTimestamp, p.Method, p.Amount, p.Comment, p.ID, id)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment: " + err.Error()})
				return
			}
			affected, err := res.RowsAffected()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check updated payment rows: " + err.Error()})
				return
			}
			if affected == 0 {
				err := tx.QueryRow("INSERT INTO payments_monitoring (date, method, order_id, amount, comment) VALUES ($1, $2, $3, $4, $5) RETURNING id",
					paymentTimestamp, p.Method, id, p.Amount, p.Comment).Scan(&p.ID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert payment for order: " + err.Error()})
					return
				}
			}
			p.Date = paymentTimestamp
		} else {
			err := tx.QueryRow("INSERT INTO payments_monitoring (date, method, order_id, amount, comment) VALUES ($1, $2, $3, $4, $5) RETURNING id",
				paymentTimestamp, p.Method, id, p.Amount, p.Comment).Scan(&p.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert new payment: " + err.Error()})
				return
			}
			p.Date = paymentTimestamp
		}
		keptPaymentIDs[p.ID] = struct{}{}
	}

	for existingID := range existingPaymentIDs {
		if _, keep := keptPaymentIDs[existingID]; keep {
			continue
		}
		if _, err := tx.Exec("DELETE FROM payments_monitoring WHERE id = $1 AND order_id = $2", existingID, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete stale payment: " + err.Error()})
			return
		}
	}

	dbPaymentsRows, err := tx.Query("SELECT id, date, method, amount, comment FROM payments_monitoring WHERE order_id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read payments for debt calculation: " + err.Error()})
		return
	}
	var dbPayments []models.Payment
	for dbPaymentsRows.Next() {
		var pm models.Payment
		if err := dbPaymentsRows.Scan(&pm.ID, &pm.Date, &pm.Method, &pm.Amount, &pm.Comment); err != nil {
			dbPaymentsRows.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse payments for debt calculation: " + err.Error()})
			return
		}
		dbPayments = append(dbPayments, pm)
	}
	dbPaymentsRows.Close()

	order.Quantity = len(order.Components)
	totalOrderAmount, err := countOrderAmountByProductIDs(tx, productIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate order amount: " + err.Error()})
		return
	}
	order.Debt = totalOrderAmount - totalPaid(dbPayments)

	_, err = tx.Exec(
		`UPDATE orders
		SET name = $1, quantity = $2, status = $3, description = $4, debt = $5,
			ship_date = $6, city = $7, full_name = $8, phone = $9, passport_inn = $10, tk = $11, places = $12, price = $13, weight = $14
		WHERE id = $15`,
		order.Name, order.Quantity, order.Status, order.Description, order.Debt,
		order.ShipDate, order.City, order.FullName, order.Phone, order.PassportInn, order.TK, order.Places, order.Price, order.Weight,
		id,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order: " + err.Error()})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}
	order.ID = id
	order.Payments = dbPayments
	c.JSON(http.StatusOK, gin.H{
		"message": "Order updated",
		"order":   order,
	})
}

func DeleteOrder(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	tx, err := DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction: " + err.Error()})
		return
	}
	defer tx.Rollback()

	var productIDs []int
	rows, err := tx.Query("SELECT product_id FROM order_products WHERE order_id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch components: " + err.Error()})
		return
	}
	for rows.Next() {
		var pid int
		if err := rows.Scan(&pid); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		productIDs = append(productIDs, pid)
	}
	rows.Close()

	for _, pid := range productIDs {
		_, err = tx.Exec("UPDATE products SET status = 1 WHERE id = $1", pid)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset product status: " + err.Error()})
			return
		}
	}

	_, err = tx.Exec("DELETE FROM orders WHERE id = $1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = tx.Commit()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order deleted"})
}
