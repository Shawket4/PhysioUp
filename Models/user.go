package Models

import (
	"PhysioUp/Utils/Token"
	"errors"
	"fmt"
	"html"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username      string        `gorm:"size:255;not null;unique" json:"username"`
	Password      string        `gorm:"size:255;not null;" json:"password"`
	Permission    int           `json:"permission"`
	Tokens        []DeviceToken `gorm:"foreignKey:UserID"`
	IsFrozen      bool          `json:"is_frozen"`
	ClinicGroupID uint          `json:"clinic_group_id"`
}

type DeviceToken struct {
	gorm.Model
	UserID uint
	Value  string `json:"value" gorm:"unique"`
}

func GetUserByID(uid uint) (User, error) {
	var user User

	if err := DB.First(&user, uid).Error; err != nil {
		return user, errors.New("User not found")
	}

	user.PrepareGive()

	return user, nil

}

func GetUserClinicGroupID(uid uint) (uint, error) {
	var clinic_id uint
	if err := DB.Model(&User{}).Where("id = ?", uid).Select("clinic_group_id").First(&clinic_id).Error; err != nil {
		return 0, errors.New("Clinic group not found")
	}

	return clinic_id, nil
}

func GetFCMsByID(uid uint) ([]string, error) {
	var fcms []string
	if err := DB.Model(&DeviceToken{}).Where("user_id = ?", uid).Select("value").Find(&fcms).Error; err != nil {
		return []string{}, errors.New("No FCMS found")
	}

	return fcms, nil
}

func GetGroupFCMsByID(uid uint) ([]string, error) {
	var fcms []string

	// First, get the clinic group ID from the user
	var user User
	if err := DB.First(&user, uid).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Find all users in the same clinic group
	var users []User
	if err := DB.Where("clinic_group_id = ?", user.ClinicGroupID).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("failed to find users in clinic group: %w", err)
	}

	// Create a map to ensure uniqueness of FCM tokens
	uniqueFCMs := make(map[string]struct{})

	// Collect all device tokens for the users
	for _, groupUser := range users {
		var tokens []DeviceToken
		if err := DB.Where("user_id = ?", groupUser.ID).Find(&tokens).Error; err != nil {
			return nil, fmt.Errorf("failed to find tokens for user %d: %w", groupUser.ID, err)
		}

		// Add tokens to the unique map
		for _, token := range tokens {
			uniqueFCMs[token.Value] = struct{}{}
		}
	}

	// Convert the map keys to a slice
	for token := range uniqueFCMs {
		fcms = append(fcms, token)
	}

	return fcms, nil
}

func (user *User) ChangeState() {
	user.IsFrozen = !user.IsFrozen
}

func (user *User) PrepareGive() {
	user.Password = ""
}

func VerifyPassword(password, hashedPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

func LoginCheck(username string, password string) (uint, string, error) {

	var err error

	user := User{}

	err = DB.Model(User{}).Where("username = ?", username).Take(&user).Error

	if err != nil {
		return 0, "", err
	}

	err = VerifyPassword(password, user.Password)

	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword {
		return 0, "", err
	}

	token, err := Token.GenerateToken(user.ID)

	if err != nil {
		return 0, "", err
	}

	return user.ID, token, nil

}

func (user *User) SaveUser() (*User, error) {

	if err := user.BeforeSave(); err != nil {
		return &User{}, err
	}

	if err := DB.Create(&user).Error; err != nil {
		return &User{}, err
	}

	return user, nil
}

func (user *User) BeforeSave() error {

	//turn password into hash
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)

	//remove spaces in username
	user.Username = html.EscapeString(strings.TrimSpace(user.Username))

	return nil

}
