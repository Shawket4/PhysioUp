package Models

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ClinicGroupExists(id uint) (bool, error) {
	var count int64
	err := DB.Model(&ClinicGroup{}).Where("id = ?", id).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func ConnectDataBase() {

	err := godotenv.Load(".env")

	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	DbHost := os.Getenv("DB_HOST")
	DbUser := os.Getenv("DB_USER")
	DbPassword := os.Getenv("DB_PASSWORD")
	DbName := os.Getenv("DB_NAME")
	DbPort := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable", DbHost, DbUser, DbPassword, DbName, DbPort)
	_ = dsn
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})

	if err != nil {
		fmt.Println("Cannot connect to database ")
		log.Fatal("connection error:", err)
	} else {
		fmt.Println("We are connected to the database ")
	}
	// First migrate models with no dependencies
	DB.AutoMigrate(&ClinicGroup{})
	DB.AutoMigrate(&SuperTreatmentPlan{})
	DB.AutoMigrate(&DeviceToken{})

	// Then migrate models that depend on the above
	DB.AutoMigrate(&User{})
	DB.AutoMigrate(&Patient{})
	DB.AutoMigrate(&Therapist{})

	// Then migrate models that depend on the previous ones
	DB.AutoMigrate(&TreatmentPlan{})
	DB.AutoMigrate(&Schedule{})

	// Finally migrate models that depend on multiple other models
	DB.AutoMigrate(&TimeBlock{})
	DB.AutoMigrate(&Referral{})
	DB.AutoMigrate(&AppointmentRequest{})
	DB.AutoMigrate(&Appointment{})
	// var plan SuperTreatmentPlan = SuperTreatmentPlan{Description: "One Organ - 6 Sessions", SessionsCount: 6}
	// DB.Save(&plan)
	// DB.AutoMigrate(&DoctorWorkingHour{})
	DB.Session(&gorm.Session{FullSaveAssociations: true})
}
