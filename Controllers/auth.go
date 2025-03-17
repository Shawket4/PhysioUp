package Controllers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"PhysioUp/Models"
	"PhysioUp/Utils/Token"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	jwt "github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

func CurrentUser(c *gin.Context) {
	user_id, err := Token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := Models.GetUserByID(user_id)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var output struct {
		ID            uint   `json:"ID"`
		Username      string `json:"username"`
		ClinicName    string `json:"clinic_name"`
		Permission    int    `json:"permission"`
		ClinicGroupID uint   `json:"clinic_group_id"`
	}
	// if user.Permission == 1 {
	// 	var doctor Models.Doctor
	// 	if err := Models.DB.Model(&Models.Doctor{}).Where("user_id = ?", user.ID).Find(&doctor).Error; err != nil {
	// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 		return
	// 	}
	// 	output.ClinicName = doctor.ClinicName
	// }
	output.ID = user_id
	output.Username = user.Username
	output.Permission = user.Permission
	output.ClinicGroupID = user.ClinicGroupID
	c.JSON(http.StatusOK, gin.H{"message": "success", "data": output})
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func Logout(c *gin.Context) {
	token, err := Token.ExtractJWT(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	claims := jwt.MapClaims{}
	claims["authorized"] = false
	claims["exp"] = time.Now()
	token2 := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Claims = token2.Claims
}

func SaveFCM(c *gin.Context) {
	var input struct {
		Token string `json:"token"`
	}
	user_id, err := Token.ExtractTokenID(c)
	if err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		// return
	}
	deviceToken := Models.DeviceToken{UserID: user_id, Value: input.Token}
	if err := Models.DB.Save(&deviceToken).Error; err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		// return
	}
	c.JSON(http.StatusOK, nil)
}

func Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := Models.User{}

	user.Username = input.Username
	user.Password = input.Password

	uid, token, err := Models.LoginCheck(user.Username, user.Password)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "username or password is incorrect."})
		return
	}

	user, _ = Models.GetUserByID(uid)

	if user.IsFrozen {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "User Frozen"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Login Successful", "jwt": token, "permission": user.Permission})

}

type RegisterInput struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	Permission    int    `json:"permission"`
	ClinicGroupID uint   `json:"clinic_group_id"`
}

func Register(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := Models.User{}

	user.Username = input.Username
	user.Password = input.Password
	user.Permission = input.Permission
	user.ClinicGroupID = input.ClinicGroupID // Don't forget to set this field
	_, err := user.SaveUser()

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": "validated"})
}

func RegisterClinicGroup(c *gin.Context) {
	var input struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	var group Models.ClinicGroup

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	group.Name = input.Name

	if err := Models.DB.Create(&group).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := Models.User{}

	user.Username = input.Name
	user.Password = input.Password
	user.Permission = 3
	user.ClinicGroupID = group.ID
	_, err := user.SaveUser()
	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "Failed To Register User")
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": "validated"})
}

func RegisterTherapist(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindBodyWith(&input, binding.JSON); err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "Bad Request")
		c.Abort()
		return
	}

	user_id, _ := Token.ExtractTokenID(c)

	clinic_group_id, err := Models.GetUserClinicGroupID(user_id)
	if err != nil {
		log.Println(err)
	}
	input.ClinicGroupID = clinic_group_id
	if input.ClinicGroupID != 0 {
		exists, err := Models.ClinicGroupExists(input.ClinicGroupID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check clinic group"})
			return
		}
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Clinic group ID does not exist"})
			return
		}
	}

	user := Models.User{}

	user.Username = input.Username
	user.Password = input.Password
	user.Permission = 2
	user.ClinicGroupID = input.ClinicGroupID
	_, err = user.SaveUser()

	if err != nil {
		log.Println(err)
		c.String(http.StatusBadRequest, "Failed To Register User")
		c.Abort()
		return
	}

	var therapist Models.Therapist

	if err := c.ShouldBindBodyWith(&therapist, binding.JSON); err != nil {
		log.Println(err.Error())
		c.JSON(http.StatusBadRequest, err)
		return
	}
	therapist.UserID = user.ID
	therapist.Schedule = Models.Schedule{TherapistID: therapist.UserID}
	therapist.Name = "Dr. " + input.Username
	// Models.CreateDoctorWorkingHours(&doctor)
	if err := Models.DB.Model(&Models.Therapist{}).Create(&therapist).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Registered Successfully"})
}

func DeleteTherapist(c *gin.Context) {
	// Extract therapist ID from the request (e.g., from URL parameters or JSON body)
	var input struct {
		ID uint `json:"id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Therapist not found"})
		return
	}

	var therapist Models.Therapist
	if err := Models.DB.Where("id = ?", input.ID).First(&therapist).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Therapist not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve therapist"})
		}
		return
	}

	var user Models.User
	if err := Models.DB.Where("id = ?", therapist.UserID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Associated user not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user"})
		}
		return
	}

	tx := Models.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Delete(&therapist).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete therapist"})
		return
	}

	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Therapist and associated user deleted successfully"})
}

func DeleteUser(c *gin.Context) {
	user_id, err := Token.ExtractTokenID(c)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := Models.GetUserByID(user_id)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := Models.DB.Unscoped().Delete(&Models.DeviceToken{}, "user_id = ?", user.ID).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := Models.DB.Unscoped().Delete(&Models.User{}, user.ID).Error; err != nil {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Account Deleted Successfully"})
}
