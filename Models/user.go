package Models

import (
	"errors"
	"html"
	"strings"

	"PhysioUp/Utils/Token"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username   string        `gorm:"size:255;not null;unique" json:"username"`
	Password   string        `gorm:"size:255;not null;" json:"password"`
	Permission int           `json:"permission"`
	Tokens     []DeviceToken `gorm:"foreignKey:UserID"`
	IsFrozen   bool          `json:"is_frozen"`
}

type DeviceToken struct {
	gorm.Model
	UserID uint
	Value  string `json:"value"`
}

func GetUserByID(uid uint) (User, error) {
	var user User

	if err := DB.Preload("Tokens").First(&user, uid).Error; err != nil {
		return user, errors.New("User not found")
	}

	user.PrepareGive()

	return user, nil

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
