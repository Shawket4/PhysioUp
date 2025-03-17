package Controllers

import (
	"PhysioUp/Models"
	"net/http"

	"github.com/gin-gonic/gin"
)

func FetchReferrals(c *gin.Context) {
	db := getScopedDB(c)
	var output []Models.Referral
	if err := db.Model(&Models.Referral{}).Find(&output).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, output)
}

func AddReferral(c *gin.Context) {
	var input Models.Referral
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client_group_id, exists := c.Get("clinicGroupID")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: Client Group Not Set"})
		return
	}

	input.ClinicGroupID = client_group_id.(uint)

	if err := Models.DB.Create(&input).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Referral Created Successfully",
	})
}

func EditReferral(c *gin.Context) {
	var input Models.Referral
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := Models.DB.Save(&input).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Referral Edited Successfully",
	})
}

func DeleteReferral(c *gin.Context) {
	var input struct {
		ReferralID uint `json:"referral_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := Models.DB.Delete(&Models.Referral{}, "id = ?", input.ReferralID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Referral Deleted Successfully",
	})
}

func FetchReferralPackages(c *gin.Context) {
	var input struct {
		ReferralID uint `json:"referral_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var TreatmentPlans []Models.TreatmentPlan
	if err := Models.DB.Model(&Models.TreatmentPlan{}).Where("referral_id = ?", input.ReferralID).Find(&TreatmentPlans).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for index := range TreatmentPlans {
		if err := Models.DB.Model(&Models.SuperTreatmentPlan{}).Where("id = ?", TreatmentPlans[index].SuperTreatmentPlanID).Find(&TreatmentPlans[index].SuperTreatmentPlan).Error; err != nil {
			c.JSON(http.StatusOK, nil)
			return
		}
	}

	c.JSON(http.StatusOK, TreatmentPlans)
}

func SetPackageReferral(c *gin.Context) {
	var input struct {
		ReferralID *uint   `json:"referral_id"`
		PackageID  uint    `json:"package_id"`
		Discount   float64 `json:"discount"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var TreatmentPlan Models.TreatmentPlan

	if err := Models.DB.Model(&Models.TreatmentPlan{}).Where("id = ?", input.PackageID).First(&TreatmentPlan).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := Models.DB.Model(&Models.SuperTreatmentPlan{}).Where("id = ?", TreatmentPlan.SuperTreatmentPlanID).Find(&TreatmentPlan.SuperTreatmentPlan).Error; err != nil {
		c.JSON(http.StatusOK, nil)
		return
	}

	TreatmentPlan.ReferralID = input.ReferralID

	TreatmentPlan.TotalPrice = TreatmentPlan.SuperTreatmentPlan.Price * ((100 - input.Discount) / 100)
	if err := Models.DB.Save(&TreatmentPlan).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Package Referral Set",
	})
}
