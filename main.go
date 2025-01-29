package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Глобальные переменные
var (
	db       *gorm.DB
	err      error
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	clients   = make(map[*websocket.Conn]bool)
	broadcast = make(chan SpotUpdate)
)

// SpotUpdate структура для обновлений свободных мест
type SpotUpdate struct {
	ParkingID uint `json:"parking_id"`
	Available int  `json:"available"`
}

func main() {
	err = godotenv.Load()
	if err != nil {
		log.Println("Нет .env файла, используются переменные окружения системы")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("Переменная DATABASE_URL не установлена")
	}
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Не удалось подключиться к базе данных:", err)
	}

	err = db.AutoMigrate(&Parking{}, &Spot{}, &Tariff{}, &Entry{}, &Exit{}, &Vehicle{}, &User{}, &Payment{})
	if err != nil {
		log.Fatal("Не удалось выполнить миграции:", err)
	}

	router := gin.Default()
	router.POST("/register", Register)
	router.POST("/login", Login)
	router.GET("/ws", WebSocketHandler)

	authorized := router.Group("/")
	authorized.Use(AuthMiddleware())
	{
		authorized.POST("/parkings", CreateParking)
		authorized.GET("/parkings", GetParkings)
		authorized.GET("/parkings/:id", GetParking)
		authorized.GET("/parkings/:id/spots", GetSpots)
		authorized.POST("/parkings/:id/spots", AddSpot)
		authorized.POST("/entries", CreateEntry)
		authorized.POST("/exits", CreateExit)
		authorized.GET("/analytics", GetAnalytics)
		authorized.POST("/payments", ProcessPayment)
	}

	go handleMessages()

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	srv := &http.Server{
		Addr:           ":" + port,
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Printf("Сервер запущен на порту %s", port)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("Ошибка запуска сервера:", err)
	}
}

// handleMessages обрабатывает отправку обновлений через WebSocket
func handleMessages() {
	for {
		update := <-broadcast

		for client := range clients {
			err := client.WriteJSON(update)
			if err != nil {
				log.Printf("Ошибка отправки сообщения клиенту: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}
