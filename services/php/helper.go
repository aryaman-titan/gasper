package php

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

	validator "github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
	"github.com/sdslabs/SWS/lib/api"
	"github.com/sdslabs/SWS/lib/configs"
	"github.com/sdslabs/SWS/lib/docker"
	"github.com/sdslabs/SWS/lib/types"
	"github.com/sdslabs/SWS/lib/utils"
)

type context struct {
	Index string `json:"index" valid:"required"`
}

type phpRequestBody struct {
	Name         string                 `json:"name" valid:"required,alphanum,stringlength(3|40)"`
	URL          string                 `json:"url" valid:"required,url"`
	Context      context                `json:"context" valid:"required"`
	Composer     bool                   `json:"composer"`
	ComposerPath string                 `json:"composerPath"`
	Env          map[string]interface{} `json:"env"`
}

func validateRequest(c *gin.Context) {

	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = ioutil.ReadAll(c.Request.Body)
	}
	// Restore the io.ReadCloser to its original state
	c.Request.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
	var req phpRequestBody

	err := json.Unmarshal(bodyBytes, &req)
	if err != nil {
		c.AbortWithStatusJSON(400, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	if result, err := validator.ValidateStruct(req); !result {
		c.AbortWithStatusJSON(400, gin.H{
			"error": err,
		})
	} else {
		c.Next()
	}
}

// installPackages installs dependancies for the specific microservice
func installPackages(path string, appEnv *types.ApplicationEnv) (string, types.ResponseError) {
	cmd := []string{"bash", "-c", `composer install -d ` + path}
	execID, err := docker.ExecDetachedProcess(appEnv.Context, appEnv.Client, appEnv.ContainerID, cmd)
	if err != nil {
		return "", types.NewResErr(500, "Failed to perform composer install in the container", err)
	}
	return execID, nil
}

func pipeline(data map[string]interface{}) types.ResponseError {
	appConf := &types.ApplicationConfig{
		DockerImage:  utils.ServiceConfig["php"].(map[string]interface{})["image"].(string),
		ConfFunction: configs.CreatePHPContainerConfig,
	}

	appEnv, resErr := api.SetupApplication(appConf, data)
	if resErr != nil {
		return resErr
	}

	// Perform composer install in the container
	if data["composer"] != nil {
		if data["composer"].(bool) {
			var composerPath string
			if data["composerPath"] != nil {
				composerPath = data["composerPath"].(string)
			} else {
				composerPath = "."
			}
			execID, resErr := installPackages(composerPath, appEnv)
			if resErr != nil {
				return resErr
			}
			data["execID"] = execID
		}
	}

	return nil
}
