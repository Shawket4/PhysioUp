package Middleware

import (
	"fmt"
	"net/http"

	"PhysioUp/Models"
	"PhysioUp/Utils/Token"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func JwtAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		err := Token.TokenValid(c)
		if err != nil {
			c.String(http.StatusUnauthorized, "Unauthorized Token Invalid")
			c.Abort()
			return
		}
		c.Next()
	}
}

func SetClinicGroup() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the user ID from the token
		userID, err := Token.ExtractTokenID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Retrieve the user from the database
		user, err := Models.GetUserByID(userID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		fmt.Println("User's ClinicGroupID:", user.ClinicGroupID)

		// Store both the original DB and the clinic group ID
		c.Set("clinicGroupID", user.ClinicGroupID)

		// Create a custom wrapper function to apply filtering
		dbWrapper := func(tableName string) *gorm.DB {
			fmt.Println("TableName: " + tableName)
			if tableName == "" {
				return Models.DB.Where("clinic_group_id = ?", user.ClinicGroupID)
			}
			return Models.DB.Where(fmt.Sprintf("%s.clinic_group_id = ?", tableName), user.ClinicGroupID)
		}

		c.Set("db", dbWrapper)
		c.Next()
	}
}

func PermissionCheckAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		user_id, err := Token.ExtractTokenID(c)

		if err != nil {
			c.String(http.StatusBadRequest, "Unauthorized Token Extraction")
			c.Abort()
			return
		}

		user, err := Models.GetUserByID(user_id)
		if err != nil {
			c.String(http.StatusBadRequest, "Unauthorized User Extraction")
			c.Abort()
			return
		}

		if user.Permission >= 2 {
			c.Next()
		} else {
			c.String(http.StatusBadRequest, "Unauthorized Not Enough Permission")
			c.Abort()
		}
	}
}

type RegisterInput struct {
	Username   string `json:"name" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Permission int    `json:"permission"`
}
