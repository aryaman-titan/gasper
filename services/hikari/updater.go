package hikari

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sdslabs/gasper/configs"
	"github.com/sdslabs/gasper/lib/redis"
	"github.com/sdslabs/gasper/lib/utils"
	"github.com/sdslabs/gasper/types"
)

func handleError(err error) {
	utils.Log("Failed to update DNS Record Storage", utils.ErrorTAG)
	utils.LogError(err)
}

// filterValidInstances filters the instances and returns
// valid instances i.e which is in the form of IP:Port
func filterValidInstances(reverseProxyInstances []string) []string {
	filteredInstances := make([]string, 0)
	for _, instance := range reverseProxyInstances {
		if strings.Contains(instance, ":") {
			filteredInstances = append(filteredInstances, instance)
		} else {
			utils.LogError(fmt.Errorf("Instance %s is of invalid format", instance))
		}
	}
	return filteredInstances
}

// Updates the DNS record storage periodically
// It assigns the A records in such a way that the load is
// equally distributed among all available Enrai Reverse Proxy Instances
func updateStorage() {
	appMap, err := redis.FetchAllApps()
	if err != nil {
		handleError(err)
		return
	}
	apps := utils.GetMapKeys(appMap)
	sort.Strings(apps)

	reverseProxyInstances, err := redis.FetchServiceInstances(types.Enrai)
	if err != nil {
		handleError(err)
		return
	}

	reverseProxyInstances = filterValidInstances(reverseProxyInstances)
	if len(reverseProxyInstances) == 0 {
		utils.Log("No valid Enrai instances available", utils.ErrorTAG)
		return
	}

	sort.Strings(reverseProxyInstances)
	updateBody := make(map[string]string)
	instanceNum := len(reverseProxyInstances)

	// Create enrties for applications
	for index, app := range apps {
		fqdn := fmt.Sprintf("%s.app.%s.", app, configs.GasperConfig.Domain)
		address := strings.Split(reverseProxyInstances[index%instanceNum], ":")[0]
		updateBody[fqdn] = address
	}

	// Create entry for Kaze
	kazeInstances, err := redis.FetchServiceInstances(types.Kaze)
	if err != nil || len(kazeInstances) == 0 {
		utils.Log(utils.InfoTAG, "No Kaze instances available. Failed to create a record for the same.")
	} else {
		kazeInstances = filterValidInstances(kazeInstances)
		for index := range kazeInstances {
			fqdn := fmt.Sprintf("%s.%s.", types.Kaze, configs.GasperConfig.Domain)
			address := strings.Split(reverseProxyInstances[index%instanceNum], ":")[0]
			updateBody[fqdn] = address
		}
	}

	storage.Replace(updateBody)
}

// ScheduleUpdate runs updateStorage on given intervals of time
func ScheduleUpdate() {
	interval := configs.ServiceConfig.Hikari.RecordUpdateInterval * time.Second
	scheduler := utils.NewScheduler(interval, updateStorage)
	scheduler.RunAsync()
}
