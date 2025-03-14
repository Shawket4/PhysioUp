package Controllers

import (
	"log"
	"net/http"
	"time"

	"PhysioUp/Models"
	"PhysioUp/Utils/Token"

	"github.com/gin-gonic/gin"
)

func GetTherapistSchedule(c *gin.Context) {
	user_id, err := Token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var therapist Models.Therapist

	if err := Models.DB.Model(&Models.Therapist{}).Where("user_id = ?", user_id).Preload("Schedule.TimeBlocks.Appointment").First(&therapist).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, therapist)
}

func AddTherapistTimeBlocks(c *gin.Context) {
	var input struct {
		DateTimes []string `json:"date_times"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	user_id, err := Token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var therapist Models.Therapist

	if err := Models.DB.Model(&Models.Therapist{}).Where("user_id = ?", user_id).Preload("Schedule.TimeBlocks").First(&therapist).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}
	for _, dateTime := range input.DateTimes {
		timeBlock := Models.CreateEmptyTimeBlock(therapist.Schedule, dateTime)
		therapist.Schedule.TimeBlocks = append(therapist.Schedule.TimeBlocks, timeBlock)
	}

	if err := Models.DB.Model(&therapist.Schedule).Where("id = ?", therapist.Schedule.ID).Association("TimeBlocks").Replace(&therapist.Schedule.TimeBlocks); err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, err)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Requested Successfully",
	})
}

func GetTherapists(c *gin.Context) {
	var therapists []Models.Therapist
	if err := Models.DB.Model(&Models.Therapist{}).Preload("Schedule.TimeBlocks.Appointment").Find(&therapists).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, therapists)
}

func GetTherapistsTrimmed(c *gin.Context) {
	var therapists []Models.Therapist

	// Get current date only in your format "yyyy/MM/dd"
	currentDate := time.Now().Format("2006/01/02")

	if err := Models.DB.Model(&Models.Therapist{}).
		// Filter by date portion only, including all times from today and future dates
		Preload("Schedule.TimeBlocks", "substr(date_time, 1, 10) >= ?", currentDate).
		Preload("Schedule.TimeBlocks.Appointment").
		Find(&therapists).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, therapists)
}
