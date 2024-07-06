package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ServerInfo struct {
	id        string
	ipAddress string
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
func hivelocityApiProvisionOrReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken string) {

	// TODO: Make more robust, check region availability before provisioning with /inventory/product/<productid> endpoint.
	// Also check out /product/<productid>/store endpoint.

	isBeingReset := (instanceId != "")

	client := &http.Client{}
	header := hivelocityGetHeaders(hveApiKey)

	if isBeingReset {
		fmt.Println("Shutting down!")
		// Returns true if the server is shutdown.
		shutdownOrCheckPower := func(doShutdown bool) bool {
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
				panic(err)
			}

			req.Header = header
			res, err := client.Do(req)

			// Check if the machine is down from api.
			data := make(map[string]interface{})

			bytes, err := io.ReadAll(res.Body)
			if err != nil {
				panic(err)
			}

			err = json.Unmarshal(bytes, &data)
			if err != nil {
				panic(err)
			}

			status, ok := data["powerStatus"]
			if !ok {
				fmt.Println("Missing permission to shutdown server.")
				fmt.Println(data)
			}

			if status == "OFF" {
				return false
			} else {
				return true
			}
		}

		const ATTEMPT_MAX_TRIES = 20
		const ATTEMPT_MAX_WAIT_TIME = time.Millisecond * 800

		// Try shutting down.
		poweredOn := shutdownOrCheckPower(true)

		for i := 0; i < ATTEMPT_MAX_TRIES; i++ {
			poweredOn = shutdownOrCheckPower(false)

			if !poweredOn {
				break
			}

			// Need to wait between attempts.
			time.Sleep(time.Millisecond * 500)
		}

		if !poweredOn {
			// Good stuff.
			fmt.Println("Succesfully shut machine down.")
		} else {
			// Not good, there's some issue.
			fmt.Println("Failed to shutdown machine. Max timeout exceeded or ")
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
		panic(err)
	}

	res, err := client.Do(req)

	if err != nil {
		panic(err)
	} else {
		val, err := io.ReadAll(res.Body)
		if err != nil {
			panic(err)
		}

		fmt.Printf("res.Body: %v\n", string(val))
	}
}

func hivelocityApiProvision(hveApiKey, xnodeId, xnodeAccessToken string) {
	// NOTE: In terms of Hivelocity's api this is the same as provisioning fresh.
	// XXX: Re-enable later, disabling because I don't want to spend any money.
	// Make a stubbed version that just cycles over known instances?
	return

	hivelocityApiProvisionOrReset(hveApiKey, "", xnodeId, xnodeAccessToken)
}

func hivelocityApiReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken string) {
	hivelocityApiProvisionOrReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken)
}

func hivelocityApiInfo(hveApiKey, instanceId string) {
	client := &http.Client{}
	header := hivelocityGetHeaders(hveApiKey)

	req, err := http.NewRequest("GET", "https://core.hivelocity.net/api/v2/device/"+instanceId, nil)
	if err != nil {
		panic(err)
	}

	req.Header = header

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	data := make(map[string]interface{})
	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(body, &data)
	if err != nil {
		panic(err)
	}
}
