package Controllers

import (
	"PhysioUp/Models"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func FetchSuperTreatments(c *gin.Context) {
	var SuperTreatments []Models.SuperTreatmentPlan
	if err := Models.DB.Model(&Models.SuperTreatmentPlan{}).Find(&SuperTreatments).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, SuperTreatments)
}

func AddSuperTreatment(c *gin.Context) {
	var input Models.SuperTreatmentPlan
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format: " + err.Error()})
		return
	}

	// #2 - Data validation
	if err := validateSuperTreatmentPlan(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fmt.Println(input)
	if err := Models.DB.Create(&input).Error; err != nil {
		// #4 - More specific error handling
		if strings.Contains(err.Error(), "foreign key constraint") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Foreign key constraint failed. Please ensure all referenced entities exist."})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create package: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Package Created Successfully",
		"id":      input.ID,
	})
}

func EditSuperTreatment(c *gin.Context) {
	var input Models.SuperTreatmentPlan
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format: " + err.Error()})
		return
	}

	// #2 - Data validation
	if err := validateSuperTreatmentPlan(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// #10 - Check if package exists
	var existingPlan Models.SuperTreatmentPlan
	if err := Models.DB.First(&existingPlan, input.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Package with ID %d not found", input.ID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check package existence: " + err.Error()})
		}
		return
	}

	// #8 - Handle referential integrity for TreatmentPlans
	// Get existing treatment plans
	var existingTreatmentPlans []Models.TreatmentPlan
	if err := Models.DB.Where("super_treatment_plan_id = ?", input.ID).Find(&existingTreatmentPlans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve associated treatment plans: " + err.Error()})
		return
	}

	// Check if changing sessions count would affect existing treatment plans
	if existingPlan.SessionsCount != input.SessionsCount && len(existingTreatmentPlans) > 0 {
		// Option 1: Update existing treatment plans (done in a transaction)
		tx := Models.DB.Begin()
		for _, plan := range existingTreatmentPlans {
			if plan.Remaining > input.SessionsCount {
				plan.Remaining = input.SessionsCount
				if err := tx.Save(&plan).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update associated treatment plans: " + err.Error()})
					return
				}
			}
		}
		tx.Commit()
	}

	// Update the super treatment plan
	if err := Models.DB.Save(&input).Error; err != nil {
		// #4 - More specific error handling
		if strings.Contains(err.Error(), "foreign key constraint") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Foreign key constraint failed. Please ensure all referenced entities exist."})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to update package: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Package Edited Successfully",
	})
}

func DeleteSuperTreatment(c *gin.Context) {
	var input struct {
		PackageID uint `json:"package_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input format: " + err.Error()})
		return
	}

	// #10 - Check if package exists before deleting
	var existingPlan Models.SuperTreatmentPlan
	if err := Models.DB.First(&existingPlan, input.PackageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Package with ID %d not found", input.PackageID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check package existence: " + err.Error()})
		}
		return
	}

	// #5 - Check for foreign key constraints (related TreatmentPlans)
	var count int64
	Models.DB.Model(&Models.TreatmentPlan{}).Where("super_treatment_plan_id = ?", input.PackageID).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Cannot delete package with ID %d. It has %d associated treatment plans.", input.PackageID, count),
		})
		return
	}

	if err := Models.DB.Delete(&Models.SuperTreatmentPlan{}, "id = ?", input.PackageID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to delete package: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Package Deleted Successfully",
	})
}

// Helper function for validation
func validateSuperTreatmentPlan(plan *Models.SuperTreatmentPlan) error {
	// #2 - Data validation
	if plan.Description == "" {
		return errors.New("description cannot be empty")
	}

	if len(plan.Description) > 255 {
		return errors.New("description is too long (maximum 255 characters)")
	}

	if plan.SessionsCount == 0 {
		return errors.New("sessions count must be greater than zero")
	}

	if plan.Price < 0 {
		return errors.New("price cannot be negative")
	}

	return nil
}
