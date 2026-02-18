package main

import (
	"github.com/gin-gonic/gin"

	"github.com/Talonmortem/SHM/db"
	"github.com/Talonmortem/SHM/internal/handlers"
	"github.com/Talonmortem/SHM/internal/middleware"
)

func main() {

	db.ConnectDB()
	defer db.CloseDB()
	db.CreateTables()
	middleware.LoadUsersRoles()

	r := gin.Default()
	r.Use(middleware.CORSMiddleware())

	// Public routes
	r.POST("/login", handlers.LoginHandler)
	r.GET("/login", handlers.LoginFormHandler)
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Protected routes
	protected := r.Group("/api").
		Use(middleware.AuthMiddleware()).
		Use(middleware.RoleMiddleware()).
		Use(middleware.RequestLogger())
	{
		protected.GET("/products/generate-name", handlers.GenerateProductNameHandler)
		protected.GET("/products", db.GetProducts)
		protected.POST("/products", db.CreateProduct)
		protected.PUT("/products/:id", db.UpdateProduct)
		protected.DELETE("/products/:id", db.DeleteProduct)
		protected.GET("/orders", db.GetOrders)
		protected.POST("/orders", db.CreateOrder)
		protected.PUT("/orders/:id", db.UpdateOrder)
		protected.DELETE("/orders/:id", db.DeleteOrder)
		protected.GET("/payment_methods", db.GetPaymentMethods)
		protected.GET("/payments_monitoring", db.GetPaymentsMonitoring)
		protected.POST("/payments", db.CreatePayment)
		protected.PUT("/payments/:id", db.UpdatePayment)
		protected.DELETE("/payments/:id", db.DeletePayment)
		protected.GET("/users", db.GetUsers)
		protected.POST("/users", db.CreateUser)
		protected.PUT("/users/:id", db.UpdateUser)
		protected.GET("/articles", db.GetArticles)
		protected.POST("/articles", db.CreateArticle)
		protected.PUT("/articles/:id", db.UpdateArticle)
		protected.DELETE("/articles/:id", db.DeleteArticle)
		protected.GET("/balance", db.GetBalance)
		protected.GET("/clients", db.GetClients)
		protected.POST("/clients", db.CreateClient)
		protected.PUT("/clients/:id", db.UpdateClient)
		protected.DELETE("/clients/:id", db.DeleteClient)
		protected.GET("/shipments", db.GetShipments)
		protected.POST("/shipments", db.CreateShipment)
		protected.PUT("/shipments/:id", db.UpdateShipment)
		protected.DELETE("/shipments/:id", db.DeleteShipment)
		protected.GET("/shipment_notes", db.GetShipmentNotes)
		protected.POST("/shipment_notes", db.CreateShipmentNote)
		protected.PUT("/shipment_notes/:id", db.UpdateShipmentNote)
		protected.DELETE("/shipment_notes/:id", db.DeleteShipmentNote)
		protected.GET("/courier_daily_payments", db.GetCourierDailyPayments)
		protected.PUT("/courier_daily_payments", db.UpsertCourierDailyPayment)
	}

	r.Run(":8086")
}
