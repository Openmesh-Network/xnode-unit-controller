package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	jp "github.com/buger/jsonparser"
)

type ServerInfo struct {
	id        string
	ipAddress string
}

func readall(readcloser io.ReadCloser) []byte {
	data, err := io.ReadAll(readcloser)
	if err != nil {
		panic(err)
	}

	return data
}

func serverInfoFromResponse(response *http.Response) ServerInfo {
	data := readall(response.Body)
	server := ServerInfo{}

	id, err := jp.GetInt(data, "deviceId")
	if err != nil {
		panic(err)
	}
	server.id = string(id)

	server.ipAddress, err = jp.GetString(data, "primaryIp")
	if err != nil {
		panic(err)
	}

	return server
}

// TODO: Make this generic, so that it can be changed from dpl without any code changes required.
func hivelocityGetCloudInitScript(xnodeId, xnodeAccessToken string) string {
	return "#cloud-config \nruncmd: \n - \"mkdir /tmp/boot && mount -t tmpfs -osize=90% none /tmp/boot && mkdir /tmp/boot/__img && wget -q -O /tmp/boot/__img/kexec.tar.xz http://boot.opnm.sh/kexec.tar.xz && mkdir /tmp/boot/system && mkdir /tmp/boot/system/proc && mount -t proc /proc /tmp/boot/system/proc && tar xvf /tmp/boot/__img/kexec.tar.xz -C /tmp/boot/system && rm /tmp/boot/__img/kexec.tar.xz && chroot /tmp/boot/system ./kexec_nixos \\\"-- XNODE_UUID=" + xnodeId + " XNODE_ACCESS_TOKEN=" + xnodeAccessToken + "\\\"\""

}

func hivelocityGetHeaders(hveApiKey string) http.Header {
	header := http.Header{}

	header.Add("X-API-KEY", hveApiKey)
	header.Add("accept", "application/json")
	header.Add("content-type", "application/json")

	return header
}

// If instanceId is "", then we provision. Otherwise, we reset.
func hivelocityApiProvisionOrReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken string) (ServerInfo, error) {

	// TODO: Make more robust, check region availability before provisioning with /inventory/product/<productid> endpoint.
	// Also check out /product/<productid>/store endpoint.

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

			if res.StatusCode != 200 {
				bytes, err := io.ReadAll(res.Body)
				if err != nil {
					panic(err)
				}

				if res.StatusCode == 400 {
					fmt.Println("Request failed, likely because we're trying to deploy while the server is reloading.")
					return false, errors.New(string(bytes))
				}

				fmt.Println("Here: ", string(bytes))

				panic("Couldn't provide server.")
				return false, errors.New("Couldn't provide server")
			} else {
				// Check if the machine is down from api.
				bytes := readall(res.Body)

				powerStatus, err := jp.GetString(bytes, "powerStatus")
				if err != nil {
					fmt.Println("Missing permission to shutdown server.")
					fmt.Println(string(bytes))
					return false, err
				}

				if powerStatus == "OFF" {
					return false, nil
				} else {
					fmt.Println(string(bytes))
					return true, nil
				}
			}
		}

		const ATTEMPT_MAX_TRIES = 20
		const ATTEMPT_MAX_WAIT_TIME = time.Millisecond * 800

		// See if it's on.
		fmt.Println("Checking if machine is off.")
		poweredOn, err := shutdownOrCheckPower(false)

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
					fmt.Println("Checking if machine is powered off. Attempt:", attempt)
					poweredOn, err = shutdownOrCheckPower(false)

					if err != nil {
						return ServerInfo{}, err
					}

					if !poweredOn {
						break
					}

					// Need to wait between attempts.
					time.Sleep(time.Millisecond * 500)
				}
			}
		}

		if !poweredOn {
			// Good stuff.
			fmt.Println("Succesfully shut machine down.")
		} else {
			// Not good, there's some issue.
			fmt.Println("Failed to shutdown machine. Max timeout exceeded.")
		}
	}

	body := map[string]interface{}{
		"osName":   "Ubuntu 22.04 (VPS)",
		"hostname": xnodeId + ".openmesh.network",
		"script":   hivelocityGetCloudInitScript(xnodeId, xnodeAccessToken),
		"tags": []string{
			"XNODE_UUID=" + xnodeId,

			// XXX: Do we want this here?
			"XNODE_ACCESS_TOKEN=" + xnodeAccessToken,
		},
	}

	requestMethod := ""
	if isBeingReset {
		requestMethod = "PUT"

		body["forceReload"] = true
	} else {
		requestMethod = "POST"

		body["period"] = "monthly"
		// XXX: Might have to change region depending on settings.
		body["locationName"] = "NYC1"

		// XXX: Change this to our product id, or load from env?
		body["productId"] = "2311"
	}

	jsonBody, err := json.Marshal(body)

	req, err := http.NewRequest(requestMethod, "https://core.hivelocity.net/api/v2/compute/"+instanceId, bytes.NewBuffer(jsonBody))
	req.Header = header

	fmt.Println("Sending request:", req.URL)

	if err != nil {
		return ServerInfo{}, err
	}

	res, err := client.Do(req)

	if err != nil {
		return ServerInfo{}, err
	} else {
		// TODO: Check code not ok!
		return serverInfoFromResponse(res), nil
	}
}

func hivelocityApiProvision(hveApiKey, xnodeId, xnodeAccessToken string) ServerInfo {
	// NOTE: In terms of Hivelocity's api this is the same as provisioning fresh.
	// XXX: Re-enable later, disabling because I don't want to spend any money.
	// Make a stubbed version that just cycles over known instances?
	fmt.Println("WARNING: Disabled provisioning for now.")
	return ServerInfo{}

	// XXX: Re-enable this.
	// hivelocityApiProvisionOrReset(hveApiKey, "", xnodeId, xnodeAccessToken)
}

func hivelocityApiReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken string) (ServerInfo, error) {
	return hivelocityApiProvisionOrReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken)
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

	if res.StatusCode != 200 {
		bytes, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}

		return ServerInfo{}, errors.New(string(bytes))
	}

	return serverInfoFromResponse(res), nil
}
