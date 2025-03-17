package Controllers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"PhysioUp/Models"
	"PhysioUp/Utils/Token"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetTherapistSchedule(c *gin.Context) {
	var input struct {
		StartDate string `json:"start_date"`
		EndDate   string `json:"end_date"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(input)

	// #13 - Date range validation
	if input.StartDate != "" && input.EndDate != "" {
		// Parse dates
		startDate, err := time.Parse("2006/01/02", input.StartDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid start date format. Use YYYY/MM/DD"})
			return
		}

		endDate, err := time.Parse("2006/01/02", input.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid end date format. Use YYYY/MM/DD"})
			return
		}

		// Ensure start date is before end date
		if startDate.After(endDate) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Start date must be before end date"})
			return
		}
	}

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

	// #5 - Check if therapist has a schedule
	var count int64
	if err := Models.DB.Model(&Models.Schedule{}).Where("therapist_id = ?", therapist.ID).Count(&count).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check schedule existence"})
		return
	}

	if count == 0 {
		// Create a new schedule for the therapist
		schedule := Models.Schedule{
			TherapistID: therapist.ID,
			TimeBlocks:  []Models.TimeBlock{},
		}
		if err := Models.DB.Create(&schedule).Error; err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule for therapist"})
			return
		}
		// Reload therapist with new schedule
		if err := Models.DB.Model(&Models.Therapist{}).Where("user_id = ?", user_id).First(&therapist).Error; err != nil {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// Fetch therapist with schedule
	if err := Models.DB.Model(&Models.Therapist{}).Where("user_id = ?", user_id).Preload("Schedule").First(&therapist).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// #12 - Use consistent approach for handling soft deletes
	// Using the built-in GORM soft delete handling instead of raw SQL
	var timeBlocks []Models.TimeBlock
	if err := Models.DB.Model(&Models.TimeBlock{}).
		Where("schedule_id = ?", therapist.Schedule.ID).
		Where("date_time >= ? AND date_time <= ?", input.StartDate, input.EndDate+" 23:59:59").
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	// #6 - Handle empty time block list
	if len(input.DateTimes) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No time blocks provided"})
		return
	}

	user_id, err := Token.ExtractTokenID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var therapist Models.Therapist
	if err := Models.DB.Model(&Models.Therapist{}).Where("user_id = ?", user_id).First(&therapist).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find therapist: " + err.Error()})
		return
	}

	// #5 - Check if therapist has a schedule and create one if not
	var schedule Models.Schedule
	if err := Models.DB.Where("therapist_id = ?", therapist.ID).First(&schedule).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create a new schedule
			schedule = Models.Schedule{
				TherapistID: therapist.ID,
				TimeBlocks:  []Models.TimeBlock{},
			}
			if err := Models.DB.Create(&schedule).Error; err != nil {
				log.Println(err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule: " + err.Error()})
				return
			}
		} else {
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check schedule: " + err.Error()})
			return
		}
	}

	// Start a transaction to ensure consistency
	tx := Models.DB.Begin()

	// List to store new time blocks
	var newTimeBlocks []Models.TimeBlock

	for _, dateTimeStr := range input.DateTimes {
		// #3 - Check for time block overlap
		var count int64
		if err := tx.Model(&Models.TimeBlock{}).
			Where("schedule_id = ? AND date_time = ?", schedule.ID, dateTimeStr).
			Count(&count).Error; err != nil {
			tx.Rollback()
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check time block overlap: " + err.Error()})
			return
		}

		if count > 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Time block already exists for %s", dateTimeStr),
			})
			return
		}

		// Create new time block
		timeBlock := Models.CreateEmptyTimeBlock(schedule, dateTimeStr)
		if err := tx.Create(&timeBlock).Error; err != nil {
			tx.Rollback()
			log.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create time block: " + err.Error()})
			return
		}

		newTimeBlocks = append(newTimeBlocks, timeBlock)
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Time blocks added successfully",
		"count":   len(newTimeBlocks),
	})
}

// TODO: Group public api
func GetTherapists(c *gin.Context) {
	user_id, _ := Token.ExtractTokenID(c)

	client_group_id, err := Models.GetUserClinicGroupID(user_id)
	if err != nil {
		log.Println(err)
	}

	query := Models.DB.Model(&Models.Therapist{}).Joins("JOIN users ON therapists.user_id = users.id").Preload("Schedule.TimeBlocks.Appointment")

	if client_group_id != 0 {
		query = query.Where("users.clinic_group_id = ?", client_group_id)
	}

	var therapists []Models.Therapist
	if err := query.Find(&therapists).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve therapists: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, therapists)
}

// TODO: Group public api
func GetTherapistsTrimmed(c *gin.Context) {

	user_id, _ := Token.ExtractTokenID(c)

	client_group_id, err := Models.GetUserClinicGroupID(user_id)
	if err != nil {
		log.Println(err)
	}

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

	query := Models.DB.Model(&Models.Therapist{}).Joins("JOIN users ON therapists.user_id = users.id").Preload("Schedule.TimeBlocks", "date_time >= ?", currentDate).
		Preload("Schedule.TimeBlocks.Appointment")

	if client_group_id == 0 {
		client_group_id = 1
	}
	query = query.Where("users.clinic_group_id = ?", client_group_id)

	if err := query.
		Find(&therapists).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch therapists"})
		return
	}

	// Convert to DTO without gorm.Model fields
	var therapistDTOs []TherapistDTO
	for _, therapist := range therapists {
		// #5 - Handle missing schedule
		if therapist.Schedule.ID == 0 {
			// Create a default empty schedule object
			therapistDTO := TherapistDTO{
				ID:       therapist.ID,
				Name:     therapist.Name,
				UserID:   therapist.UserID,
				Phone:    therapist.Phone,
				PhotoUrl: therapist.PhotoUrl,
				IsDemo:   therapist.IsDemo,
				IsFrozen: therapist.IsFrozen,
				Schedule: ScheduleDTO{
					ID:         0,
					TimeBlocks: []TimeBlockDTO{},
				},
			}
			therapistDTOs = append(therapistDTOs, therapistDTO)
			continue
		}

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
