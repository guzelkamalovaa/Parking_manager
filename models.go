package main

import (
	"time"

	"gorm.io/gorm"
)

// Парковка (Parking)
type Parking struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `json:"name"`
	Latitude  float64        `json:"latitude"`
	Longitude float64        `json:"longitude"`
	Capacity  int            `json:"capacity"`
	Tariffs   []Tariff       `json:"tariffs" gorm:"foreignKey:ParkingID;constraint:OnDelete:CASCADE"`
	Spots     []Spot         `json:"spots" gorm:"foreignKey:ParkingID;constraint:OnDelete:CASCADE"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Место на парковке (Spot)
type Spot struct {
	ID         uint           `gorm:"primaryKey" json:"id"`
	ParkingID  uint           `json:"parking_id"`
	Number     string         `json:"number"`
	IsOccupied bool           `json:"is_occupied"`
	Entries    []Entry        `json:"entries" gorm:"foreignKey:SpotID;constraint:OnDelete:SET NULL"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// Тариф (Tariff)
type Tariff struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	ParkingID uint           `json:"parking_id"`
	Type      string         `json:"type"` // Например, почасовой, дневной
	Price     float64        `json:"price"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Въезд автомобиля (Entry)
type Entry struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	SpotID    uint           `json:"spot_id"`
	VehicleID uint           `json:"vehicle_id"`
	EntryTime time.Time      `json:"entry_time"`
	ExitTime  *time.Time     `json:"exit_time,omitempty"`
	Exit      *Exit          `json:"exit,omitempty" gorm:"foreignKey:EntryID"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Выезд автомобиля (Exit)
type Exit struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	EntryID   uint           `json:"entry_id"`
	ExitTime  time.Time      `json:"exit_time"`
	PaymentID uint           `json:"payment_id"`
	Payment   Payment        `json:"payment" gorm:"foreignKey:PaymentID"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Автомобиль (Vehicle)
type Vehicle struct {
	ID           uint           `gorm:"primaryKey" json:"id"`
	LicensePlate string         `json:"license_plate" gorm:"uniqueIndex"`
	OwnerID      uint           `json:"owner_id"`
	Owner        User           `json:"owner" gorm:"foreignKey:OwnerID"`
	Entries      []Entry        `json:"entries" gorm:"foreignKey:VehicleID;constraint:OnDelete:SET NULL"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Пользователь (User)
type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Name      string         `json:"name"`
	Email     string         `json:"email" gorm:"uniqueIndex"`
	Password  string         `json:"-"` // Хранится хеш пароля
	Vehicles  []Vehicle      `json:"vehicles" gorm:"foreignKey:OwnerID;constraint:OnDelete:CASCADE"`
	Entries   []Entry        `json:"entries" gorm:"foreignKey:OwnerID;constraint:OnDelete:SET NULL"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// Платеж (Payment)
type Payment struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Amount    float64        `json:"amount"`
	Method    string         `json:"method"` // Например, кредитная карта, PayPal
	Status    string         `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
