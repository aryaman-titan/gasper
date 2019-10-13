package middlewares

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	g "github.com/gin-gonic/gin"
	"github.com/sdslabs/gasper/configs"
	falconApi "github.com/supra08/falcon-client-golang"
)

var falconConf falconApi.FalconClientGolang

// InitializeFalconConfig intializes the falcon API
func InitializeFalconConfig() {
	clientID := configs.FalconConfig.FalconClientID
	clientSecret := configs.FalconConfig.FalconClientSecret
	falconURLAccessToken := configs.FalconConfig.FalconURLAccessToken
	falconURLResourceOwner := configs.FalconConfig.FalconURLResourceOwnerDetails
	falconAccountsURL := configs.FalconConfig.FalconAccountsURL
	falconConf = falconApi.New(clientID, clientSecret, falconURLAccessToken, falconURLResourceOwner, falconAccountsURL)
}

func getUser(cookie string) (string, error) {
	if !strings.Contains(cookie, "sdslabs") {
		return "", errors.New("User not logged in")
	}
	hash := strings.Split(cookie, "=")[1]
	user, err := falconApi.GetLoggedInUser(falconConf, hash)
	if err != nil {
		return "", errors.New("error in falcon client")
	}
	return user, nil
}

// FalconGuard is a middleware for checking whether the user is logged into accounts or not
func FalconGuard() gin.HandlerFunc {
	if configs.FalconConfig.PlugIn {
		return func(c *gin.Context) {
			cookie := c.GetHeader("Cookie")
			user, err := getUser(cookie)
			if user == "" {
				c.JSON(403, gin.H{
					"success": false,
					"error":   err.Error(),
				})
				c.Abort()
				return
			}
			c.Next()
		}
	}
	return func(c *g.Context) {
		c.Next()
	}
}
