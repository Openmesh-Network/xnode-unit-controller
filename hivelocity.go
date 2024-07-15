package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"math/rand"

	jp "github.com/buger/jsonparser"
)

type ServerInfo struct {
	Id          int    `json:"deviceId"`
	IpAddress   string `json:"primaryIp"`
	Location    string `json:"locationName"`
	PowerStatus string `json:"powerStatus"`
}

func readall(readcloser io.ReadCloser) []byte {
	data, err := io.ReadAll(readcloser)
	if err != nil {
		panic(err)
	}

	return data
}

func isResponseSuccessful(response *http.Response) bool {
	if response.StatusCode >= 200 && response.StatusCode <= 299 {
		return true
	} else {
		return false
	}
}

func messageFromResponse(response *http.Response) string {
	data, err := io.ReadAll(response.Body)

	if err != nil {
		fmt.Println("Error reading response: ", err)
		return ""
	} else {
		message, err := jp.GetString(data)

		if err != nil {
			fmt.Println("Error reading response: ", err)
			return ""
		} else {
			return message
		}
	}
}

func serverInfoFromResponse(response *http.Response) ServerInfo {
	var server ServerInfo
	data, readErr := io.ReadAll(response.Body)
	if readErr != nil {
		fmt.Println("Error reading response: ", readErr)
		return ServerInfo{}
	}

	jsonErr := json.Unmarshal(data, &server)
	if jsonErr != nil {
		fmt.Println("Error reading response: ", jsonErr)
		return ServerInfo{}
	}

	return server
}

// TODO: Make this generic, so that it can be changed from dpl without any code changes required.
func hivelocityGetCloudInitKexecScript(xnodeId string, xnodeAccessToken string, xnodeConfigRemote string) string {
	init := "#cloud-config \nruncmd: \n - "
	ramfs := "\"mkdir /tmp/boot && mount -t tmpfs -osize=90% none /tmp/boot && mkdir /tmp/boot/__img && "
	getArtifact := "wget -q -O /tmp/boot/__img/kexec.tar.xz http://boot.opnm.sh/kexec.tar.xz && "
	procInit := "mkdir /tmp/boot/system && mkdir /tmp/boot/system/proc && mount -t proc /proc /tmp/boot/system/proc && "
	unpackAndKexec := "tar xvf /tmp/boot/__img/kexec.tar.xz -C /tmp/boot/system && rm /tmp/boot/__img/kexec.tar.xz && chroot /tmp/boot/system ./kexec_nixos"
	kparams := "-- XNODE_UUID=" + xnodeId + " XNODE_ACCESS_TOKEN=" + xnodeAccessToken + " XNODE_CONFIG_REMOTE=" + xnodeConfigRemote + " AVOID_NEWLINE=1 \""
	return init + ramfs + getArtifact + procInit + unpackAndKexec + kparams
}

func hivelocityGetCloudInitScript(xnodeId string, xnodeAccessToken string, xnodeConfigRemote string) string {
	init := "#cloud-config \nruncmd: \n - "
	pullXnodeAssimilate := "curl https://raw.githubusercontent.com/Openmesh-Network/XnodeOS-assimilate/dev/xnodeos-assimilate | "
	acceptDestroySystem := `ACCEPT_DESTRUCTION_OF_SYSTEM="Yes, destroy my system and delete all of my data. I know what I'm doing." `
	KernelParams := "XNODE_KERNEL_EXTRA_PARAMS=1 XNODE_UUID=" + xnodeId + " XNODE_ACCESS_TOKEN=" + xnodeAccessToken + " XNODE_CONFIG_REMOTE=" + xnodeConfigRemote
	log := ` bash 2>&1 | tee /tmp/assimilate.log`
	return init + pullXnodeAssimilate + acceptDestroySystem + KernelParams + log
}

func hivelocityGetHeaders(hveApiKey string) http.Header {
	header := http.Header{}

	header.Add("X-API-KEY", hveApiKey)
	header.Add("accept", "application/json")
	header.Add("content-type", "application/json")

	return header
}

// If instanceId is "", then we provision. Otherwise, we reset.
func hivelocityApiProvisionOrReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken, xnodeConfigRemote string) (ServerInfo, error) {
	isBeingReset := (instanceId != "")

	client := &http.Client{}
	header := hivelocityGetHeaders(hveApiKey)

	if isBeingReset {
		fmt.Println("Shutting down!")
		// Returns true if the server is shutdown.
		shutdownOrCheckPower := func(doShutdown bool) (bool, error) {
			urlCheck := "https://core.hivelocity.net/api/v2/device/" + instanceId + "/power"
			urlShutdown := urlCheck + "?action=shutdown"

			url := urlCheck
			method := "GET"

			if doShutdown {
				url = urlShutdown
				method = "POST"
			}

			fmt.Println("Making request: ", url)

			req, err := http.NewRequest(method, url, nil)
			if err != nil {
				return false, err
			}

			req.Header = header
			res, err := client.Do(req)
			if err != nil {
				panic(err)
			}

			if isResponseSuccessful(res) {
				// Check if the machine is down from api.
				info := serverInfoFromResponse(res)
				fmt.Printf("Found instance %d had power status: %s", info.Id, info.PowerStatus)
				if info.PowerStatus == "OFF" {
					return false, nil
				} else {
					fmt.Println(info)
					return true, nil
				}
			} else {
				_, err := io.ReadAll(res.Body)
				if err != nil {
					return false, errors.New("Hivelocity API didn't return a valid response body on shutdown / info request.")
				}

				if res.StatusCode == 400 {
					message := "Request failed, likely because we're trying to deploy while the server is reloading. Error response: " + messageFromResponse(res)
					return false, errors.New(message)
				} else if res.StatusCode == 403 {
					message := "Request failed, authorization invalid."
					fmt.Println(message)
					return false, errors.New(message)
				}

				fmt.Println("Code should never reach this point. This means our API key doesn't have authorization. Error response: ", messageFromResponse(res), res.StatusCode)
				return false, errors.New("Couldn't provide server.")
			}
		}

		// XXX: Took more than 30 seconds????
		const ATTEMPT_MAX_TRIES = 20
		const ATTEMPT_COOLDOWN_TIME = time.Millisecond * 1500

		fmt.Println("Checking if machine is off.")
		poweredOn, err := shutdownOrCheckPower(false)

		if err != nil {
			return ServerInfo{}, err
		}

		if poweredOn {
			fmt.Println("It's on, shutting down.")
			// If it's not try shutting down.
			poweredOn, err = shutdownOrCheckPower(true)
			if err != nil {
				return ServerInfo{}, err
			}

			fmt.Println("Sent shutdown machine command.")
			if poweredOn {
				for attempt := 0; attempt < ATTEMPT_MAX_TRIES; attempt++ {
					fmt.Println("Checking if machine is powered off. Attempt:", attempt+1)
					poweredOn, err = shutdownOrCheckPower(false)

					if err != nil {
						return ServerInfo{}, err
					}

					if !poweredOn {
						break
					}

					// Need to wait between attempts.
					time.Sleep(ATTEMPT_COOLDOWN_TIME)
				}
			}
		}

		if !poweredOn {
			// Good stuff.
			fmt.Println("Succesfully shut machine down.")
		} else {
			// Not good, there's some issue.
			return ServerInfo{}, errors.New("failed to shutdown machine. Max timeout exceeded.")
		}
	}

	body := map[string]interface{}{
		"osName":   "Ubuntu 22.04 (VPS)",
		"hostname": xnodeId + ".openmesh.network",
		"script":   hivelocityGetCloudInitScript(xnodeId, xnodeAccessToken, xnodeConfigRemote),
		"tags": []string{
			"XNODE_UUID=" + xnodeId,
			"XNODE_CONFIG_REMOTE=" + xnodeConfigRemote,

			// XXX: Do we want this here?
			// Uncommenting this means anyone with the API key can control the xnode.
			// Being cautious for now.
			// "XNODE_ACCESS_TOKEN=" + xnodeAccessToken_DELETEME,
		},
	}

	requestMethod := ""
	if isBeingReset {
		requestMethod = "PUT"

		body["forceReload"] = true
	} else {
		// Also check out /product/<productid>/store endpoint.
		requestMethod = "POST"
		body["period"] = "monthly"

		locationName, capacityError := hivelocityFirstAvailableRegion(hveApiKey, "2379")
		if (body["locationName"] == "" || capacityError != nil) && !isBeingReset {
			fmt.Println("Unable to find any available regions")
			return ServerInfo{}, errors.New("No available regions.")
		}
		body["locationName"] = locationName

		// XXX: Change this to our product id, or load from env?
		body["productId"] = "2379"
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		fmt.Println("Failed to marshal body of request. Error: ", err)
	}

	req, err := http.NewRequest(requestMethod, "https://core.hivelocity.net/api/v2/compute/"+instanceId, bytes.NewBuffer(jsonBody))
	if err != nil {
		return ServerInfo{}, err
	}
	req.Header = header
	fmt.Println("Sending request:", req.URL)

	response, err := client.Do(req)

	if err != nil {
		return ServerInfo{}, err
	}

	if isResponseSuccessful(response) {
		info := serverInfoFromResponse(response)
		fmt.Println("Got info: ", info)
		return info, nil
	} else {
		message := messageFromResponse(response)
		return ServerInfo{}, errors.New("Failed to reset or provision. Error: " + message)
	}
}

func hivelocityApiProvision(hveApiKey, xnodeId, xnodeAccessToken, xnodeConfigRemote string) (ServerInfo, error) {

	if os.Getenv("MOCK_PROVISIONING") == "1" {
		// Chose random machine and reset instead
		fmt.Println("Hack, instead of provisioning a full machine. We instead hard code an instance id to always reset")
		potentialIds := []string{}
		if os.Getenv("MOCK_DEVICES") != "" {
			fmt.Println("Using mock device ids")
			mockIds := os.Getenv("MOCK_DEVICES")
			mockIds = strings.Trim(mockIds, "[]")
			potentialIds = strings.Split(mockIds, ` `)
		} else {
			// XXX: Can get errors if any of these devices do not exist for the specified API key.
			potentialIds = []string{"39956", "39954", "39939", "39879", "39878", "39877", "39876", "39875", "39874", "39873", "39872", "39871", "39818", "39817"}
		}

		// Pick a random device id to reset.
		randomIndex := rand.Intn(len(potentialIds))
		id := potentialIds[randomIndex]
		return hivelocityApiProvisionOrReset(hveApiKey, id, xnodeId, xnodeAccessToken, xnodeConfigRemote)
	} else {
		// XXX: Hivelocity puts pending charges on a credit card if you provision anything for the account, regardless if the machine is invoiced on credit.
		//	- In the case that the card declines, Hivelocity marks the machine's status as "verification" which then pauses provisioning.
		//		Working with them to fix this in production.

		return hivelocityApiProvisionOrReset(hveApiKey, "", xnodeId, xnodeAccessToken, xnodeConfigRemote)
	}
}

func hivelocityApiReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken, xnodeConfigRemote string) (ServerInfo, error) {
	return hivelocityApiProvisionOrReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken, xnodeConfigRemote)
}

func hivelocityApiInfo(hveApiKey, instanceId string) (ServerInfo, error) {
	client := &http.Client{}
	header := hivelocityGetHeaders(hveApiKey)

	req, err := http.NewRequest("GET", "https://core.hivelocity.net/api/v2/device/"+instanceId, nil)
	if err != nil {
		panic(err)
	}

	req.Header = header

	res, err := client.Do(req)
	if err != nil {
		return ServerInfo{}, err
	}

	if isResponseSuccessful(res) {
		return serverInfoFromResponse(res), nil
	} else {
		return ServerInfo{}, errors.New(messageFromResponse(res))
	}
}

func hivelocityFirstAvailableRegion(hveApiKey string, productId string) (string, error) {
	client := &http.Client{}
	header := hivelocityGetHeaders(hveApiKey)

	req, err := http.NewRequest("GET", "https://core.hivelocity.net/api/v2/inventory/product/"+productId, nil)
	if err != nil {
		return "", err
	}

	req.Header = header

	response, err := client.Do(req)
	if err != nil {
		fmt.Println("Error in determining available regions", err)
		return "", err
	}

	if isResponseSuccessful(response) {
		var availableRegions map[string]interface{}
		json.Unmarshal(readall(response.Body), &availableRegions)

		for region := range availableRegions {
			fmt.Println("Region: ", region)
			// fmt.Println("Available: ", availableRegions)
			locations := availableRegions[region].([]interface{})

			for loc := range locations {
				locationData := locations[loc].(map[string]interface{})
				availability := locationData["stock"]
				if availability == "available" {
					fmt.Printf("Available at %s in region: %s\n", locationData["data_center"], region)
					return region, nil
				}
			}
		}

		return "", fmt.Errorf("No available regions found.")
	} else {
		err := errors.New("Request to find regions failed. Is the API key correct?")
		return "", err
	}
}
