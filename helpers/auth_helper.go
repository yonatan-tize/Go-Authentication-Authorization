package helpers

import (
	"errors"
	"github.com/gin-gonic/gin"
)

func CheckUserType(c *gin.Context, role string) error{
	userType := c.GetString("user_type")
	var err error = nil
	if userType != role {
		err = errors.New("unautorized to access this resourse")
		return err
	}
	return err
}

func MatchUserTypeToUid(c *gin.Context, userId string) error {
	userType := c.GetString("userr_type")
	uid := c.GetString("uid")
	var err error= nil

	if userType == "USER" && uid !=userId{
		err = errors.New("Unauthorized to access this resource")
		return err
	}
	err = CheckUserType(c, userType)
	return err
}
