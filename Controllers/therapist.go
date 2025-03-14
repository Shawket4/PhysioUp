package Controllers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"PhysioUp/Models"
	"PhysioUp/Utils/Token"

	"github.com/gin-gonic/gin"
)

func GetTherapistSchedule(c *gin.Context) {
	const PaginationValue = 6 // Number of weeks per page
	var input struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(input)
	user_id, err := Token.ExtractTokenID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var therapist Models.Therapist
	if err := Models.DB.Model(&Models.Therapist{}).Where("user_id = ?", user_id).First(&therapist).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Default to current month if dates are not provided
	if input.StartDate == "" || input.EndDate == "" {
		now := time.Now()
		// First day of current month
		firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		// Last day of current month
		lastDay := firstDay.AddDate(0, 1, -1)

		input.StartDate = firstDay.Format("2006/01/02")
		input.EndDate = lastDay.Format("2006/01/02")

		fmt.Println("Using default date range:", input.StartDate, "to", input.EndDate)
	}

	// Fetch therapist with schedule
	if err := Models.DB.Model(&Models.Therapist{}).Where("user_id = ?", user_id).Preload("Schedule").First(&therapist).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Use raw SQL query with BETWEEN and LIKE to handle string date format
	query := `
		SELECT tb.* FROM time_blocks tb
		WHERE tb.schedule_id = ?
		AND (
			SUBSTR(tb.date_time, 1, 10) BETWEEN ? AND ?
		)
		AND tb.deleted_at IS NULL
	`
	var timeBlocks []Models.TimeBlock
	if err := Models.DB.Raw(query, therapist.Schedule.ID, input.StartDate, input.EndDate).
		Preload("Appointment").
		Find(&timeBlocks).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Assign filtered time blocks to the schedule
	therapist.Schedule.TimeBlocks = timeBlocks

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
	// Define response structures without the gorm.Model fields
	type TimeBlockDTO struct {
		ID          uint   `json:"ID"`
		DateTime    string `json:"date"`
		IsAvailable bool   `json:"is_available"`
	}

	type ScheduleDTO struct {
		ID         uint           `json:"ID"`
		TimeBlocks []TimeBlockDTO `json:"time_blocks"`
	}

	type TherapistDTO struct {
		ID       uint        `json:"ID"`
		Name     string      `json:"name"`
		UserID   uint        `json:"user_id"`
		Phone    string      `json:"phone"`
		Schedule ScheduleDTO `json:"schedule"`
		PhotoUrl string      `json:"photo_url"`
		IsDemo   bool        `json:"is_demo"`
		IsFrozen bool        `json:"is_frozen"`
	}

	// Fetch data from database
	var therapists []Models.Therapist
	currentDate := time.Now().Format("2006/01/02")

	if err := Models.DB.Model(&Models.Therapist{}).
		Preload("Schedule.TimeBlocks", "substr(date_time, 1, 10) >= ?", currentDate).
		Preload("Schedule.TimeBlocks.Appointment").
		Find(&therapists).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch therapists"})
		return
	}

	// Convert to DTO without gorm.Model fields
	var therapistDTOs []TherapistDTO
	for _, therapist := range therapists {
		therapistDTO := TherapistDTO{
			ID:       therapist.ID,
			Name:     therapist.Name,
			UserID:   therapist.UserID,
			Phone:    therapist.Phone,
			PhotoUrl: therapist.PhotoUrl,
			IsDemo:   therapist.IsDemo,
			IsFrozen: therapist.IsFrozen,
			Schedule: ScheduleDTO{
				ID: therapist.Schedule.ID,
			},
		}

		// Add time blocks
		for _, block := range therapist.Schedule.TimeBlocks {
			blockDTO := TimeBlockDTO{
				ID:          block.ID,
				DateTime:    block.DateTime,
				IsAvailable: block.IsAvailable,
			}

			therapistDTO.Schedule.TimeBlocks = append(therapistDTO.Schedule.TimeBlocks, blockDTO)
		}

		therapistDTOs = append(therapistDTOs, therapistDTO)
	}

	c.JSON(http.StatusOK, therapistDTOs)
}
