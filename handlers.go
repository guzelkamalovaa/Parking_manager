package main

import (
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
)

type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

func Register(c *gin.Context) {
	var input struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existingUser User
	if err := db.Where("email = ?", input.Email).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Пользователь с таким email уже существует"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обработать пароль"})
		return
	}

	user := User{
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashedPassword),
	}

	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать пользователя"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Пользователь успешно зарегистрирован"})
}

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	if err := db.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверные учетные данные"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Неверные учетные данные"})
		return
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: user.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "parking_api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "JWT_SECRET не установлен"})
		return
	}

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать токен"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Токен не предоставлен"})
			return
		}

		claims := &Claims{}
		jwtSecret := os.Getenv("JWT_SECRET")
		if jwtSecret == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "JWT_SECRET не установлен"})
			return
		}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Неверный токен"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Next()
	}
}

func CreateParking(c *gin.Context) {
	var input struct {
		Name      string   `json:"name" binding:"required"`
		Latitude  float64  `json:"latitude" binding:"required"`
		Longitude float64  `json:"longitude" binding:"required"`
		Capacity  int      `json:"capacity" binding:"required,min=1"`
		Tariffs   []TariffInput `json:"tariffs" binding:"required,dive,required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	parking := Parking{
		Name:      input.Name,
		Latitude:  input.Latitude,
		Longitude: input.Longitude,
		Capacity:  input.Capacity,
	}

	for _, t := range input.Tariffs {
		parking.Tariffs = append(parking.Tariffs, Tariff{
			Type:  t.Type,
			Price: t.Price,
		})
	}

	if err := db.Create(&parking).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать парковку"})
		return
	}

	c.JSON(http.StatusCreated, parking)
}

type TariffInput struct {
	Type  string  `json:"type" binding:"required"`
	Price float64 `json:"price" binding:"required,gt=0"`
}

func GetParkings(c *gin.Context) {
	var parkings []Parking
	if err := db.Preload("Tariffs").Preload("Spots").Find(&parkings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить парковки"})
		return
	}

	c.JSON(http.StatusOK, parkings)
}

func GetParking(c *gin.Context) {
	id := c.Param("id")
	var parking Parking
	if err := db.Preload("Tariffs").Preload("Spots").First(&parking, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Парковка не найдена"})
		return
	}

	c.JSON(http.StatusOK, parking)
}

func GetSpots(c *gin.Context) {
	parkingID := c.Param("id")
	var spots []Spot
	if err := db.Where("parking_id = ?", parkingID).Find(&spots).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить места"})
		return
	}

	c.JSON(http.StatusOK, spots)
}

func AddSpot(c *gin.Context) {
	parkingID := c.Param("id")
	var input struct {
		Number string `json:"number" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var parking Parking
	if err := db.First(&parking, parkingID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Парковка не найдена"})
		return
	}

	spot := Spot{
		ParkingID:  parking.ID,
		Number:     input.Number,
		IsOccupied: false,
	}

	if err := db.Create(&spot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось добавить место"})
		return
	}

	notifySpotUpdate(parking.ID)

	c.JSON(http.StatusCreated, spot)
}

func CreateEntry(c *gin.Context) {
	var input struct {
		SpotID    uint   `json:"spot_id" binding:"required"`
		VehicleID uint   `json:"vehicle_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var spot Spot
	if err := db.First(&spot, input.SpotID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Место не найдено"})
		return
	}

	if spot.IsOccupied {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Место уже занято"})
		return
	}

	var vehicle Vehicle
	if err := db.First(&vehicle, input.VehicleID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Автомобиль не найден"})
		return
	}

	entry := Entry{
		SpotID:    spot.ID,
		VehicleID: vehicle.ID,
		EntryTime: time.Now(),
	}

	if err := db.Create(&entry).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось зафиксировать въезд"})
		return
	}

	spot.IsOccupied = true
	if err := db.Save(&spot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить статус места"})
		return
	}

	notifySpotUpdate(spot.ParkingID)

	c.JSON(http.StatusCreated, entry)
}

func CreateExit(c *gin.Context) {
	var input struct {
		EntryID        uint   `json:"entry_id" binding:"required"`
		PaymentMethod  string `json:"payment_method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var entry Entry
	if err := db.First(&entry, input.EntryID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Запись о въезде не найдена"})
		return
	}

	if entry.ExitTime != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Выезд уже зафиксирован"})
		return
	}

	amount := calculatePayment(entry.EntryTime, time.Now())

	payment := Payment{
		Amount:   amount,
		Method:   input.PaymentMethod,
		Status:   "pending",
		CreatedAt: time.Now(),
	}

	if err := db.Create(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось создать платеж"})
		return
	}

	exit := Exit{
		EntryID:   entry.ID,
		ExitTime:  time.Now(),
		PaymentID: payment.ID,
	}

	if err := db.Create(&exit).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось зафиксировать выезд"})
		return
	}

	entry.ExitTime = &exit.ExitTime
	if err := db.Save(&entry).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить запись о въезде"})
		return
	}

	var spot Spot
	if err := db.First(&spot, entry.SpotID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Место не найдено"})
		return
	}

	spot.IsOccupied = false
	if err := db.Save(&spot).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить статус места"})
		return
	}

	notifySpotUpdate(spot.ParkingID)

	c.JSON(http.StatusCreated, exit)
}

func GetAnalytics(c *gin.Context) {
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Параметры start_time и end_time обязательны"})
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат start_time"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат end_time"})
		return
	}

	var results []struct {
		ParkingID uint `gorm:"column:parking_id" json:"parking_id"`
		Count     int  `gorm:"column:count" json:"count"`
		Hour      int  `gorm:"column:hour" json:"hour"`
	}

	db.Table("entries").
		Select("parkings.id as parking_id, EXTRACT(HOUR FROM entries.entry_time) as hour, COUNT(*) as count").
		Joins("JOIN spots ON spots.id = entries.spot_id").
		Joins("JOIN parkings ON parkings.id = spots.parking_id").
		Where("entries.entry_time BETWEEN ? AND ?", startTime, endTime).
		Group("parkings.id, hour").
		Scan(&results)

	c.JSON(http.StatusOK, results)
}

func ProcessPayment(c *gin.Context) {
	var input struct {
		PaymentMethodID string `json:"payment_method_id" binding:"required"`
		Amount          int64  `json:"amount" binding:"required,gt=0"` // В копейках
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "STRIPE_SECRET_KEY не установлен"})
		return
	}

	params := &stripe.PaymentIntentParams{
		Amount:              stripe.Int64(input.Amount),
		Currency:            stripe.String(string(stripe.CurrencyRub)),
		PaymentMethod:       stripe.String(input.PaymentMethodID),
		ConfirmationMethod:  stripe.String(string(stripe.PaymentIntentConfirmationMethodAutomatic)),
		Confirm:             stripe.Bool(true),
		PaymentMethodTypes: stripe.StringSlice([]string{"card"}),
	}

	pi, err := paymentintent.New(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка при обработке платежа"})
		return
	}

	var payment Payment
	if err := db.First(&payment, pi.ID).Error; err != nil {
		payment = Payment{
			ID:     uint(pi.ID),
			Amount: float64(pi.Amount) / 100, // Переводим из копеек в рубли
			Method: "Stripe",
			Status: string(pi.Status),
		}
		if err := db.Create(&payment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось сохранить платеж"})
			return
		}
	} else {
		payment.Status = string(pi.Status)
		if err := db.Save(&payment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось обновить платеж"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"payment_id": payment.ID,
		"status":     payment.Status,
	})
}

func WebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось установить WebSocket соединение"})
		return
	}
	defer conn.Close()

	clients[conn] = true

	for {
		var msg interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			delete(clients, conn)
			break
		}
	}
}

func notifySpotUpdate(parkingID uint) {
	var available int64
	db.Model(&Spot{}).Where("parking_id = ? AND is_occupied = ?", parkingID, false).Count(&available)

	update := SpotUpdate{
		ParkingID: parkingID,
		Available: int(available),
	}

	broadcast <- update
}

func calculatePayment(entryTime, exitTime time.Time) float64 {
	duration := exitTime.Sub(entryTime)
	hours := int(duration.Hours())
	if duration.Minutes() > 0 && duration.Seconds()%3600 > 0 {
		hours += 1
	}

	var tariff Tariff
	if err := db.Where("parking_id = ?", 1).Where("type = ?", "почасовой").First(&tariff).Error; err != nil {
		return float64(hours) * 2.5
	}

	return float64(hours) * tariff.Price
}
