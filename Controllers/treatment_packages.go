package Controllers

import (
	"PhysioUp/Models"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println(input)
	if err := Models.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Package Created Successfully",
	})
}

func EditSuperTreatment(c *gin.Context) {
	var input Models.SuperTreatmentPlan
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := Models.DB.Save(&input).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := Models.DB.Delete(&Models.SuperTreatmentPlan{}, "id = ?", input.PackageID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Package Deleted Successfully",
	})
}
