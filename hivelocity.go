package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// TODO: Make this generic, so that it can be changed from dpl.
func hivelocityGetCloudInitScript(xnodeId string, xnodeAccessToken string) string {
	return "#cloud-config \nruncmd: \n - \"mkdir /tmp/boot && mount -t tmpfs -osize=90% none /tmp/boot && mkdir /tmp/boot/__img && wget -q -O /tmp/boot/__img/kexec.tar.xz http://boot.opnm.sh/kexec.tar.xz && mkdir /tmp/boot/system && mkdir /tmp/boot/system/proc && mount -t proc /proc /tmp/boot/system/proc && tar xvf /tmp/boot/__img/kexec.tar.xz -C /tmp/boot/system && rm /tmp/boot/__img/kexec.tar.xz && chroot /tmp/boot/system ./kexec_nixos \\\"-- XNODE_UUID=" + xnodeId + " XNODE_ACCESS_TOKEN=" + xnodeAccessToken + "\\\"\""

}

// If instanceId is "", then we provision. Otherwise, we reset.
func hivelocityApiProvisionOrReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken string) {

	// TODO: Make more robust, check region availability before provisioning with /inventory/product/<productid> endpoint.
	// Also check out /product/<productid>/store endpoint.

	body := map[string]interface{}{
		"osName": "Ubuntu 22.04 (VPS)",
		"hostname": xnodeId + ".openmesh.network",
		"script":   hivelocityGetCloudInitScript(xnodeId, xnodeAccessToken),
	}

	requestMethod := ""
	if instanceId == "" {
		requestMethod = "POST"

		body["period"] = "monthly"
		// XXX: Might have to change region depending on settings.
		body["locationName"] = "NYC1"

		// XXX: Change this to our product id, or load from env?
		body["productId"] = "2311"
	} else {
		requestMethod = "PUT"

		body["forceReload"] = true
	}

	jsonBody, err := json.Marshal(body)

	req, err := http.NewRequest(requestMethod, "https://core.hivelocity.net/api/v2/compute/"+instanceId, bytes.NewBuffer(jsonBody))

	if err != nil {
		panic(err)
	}

	req.Header.Add("X-API-KEY", hveApiKey)
	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")

	client := &http.Client{}
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
	return

	hivelocityApiProvisionOrReset(hveApiKey, "", xnodeId, xnodeAccessToken)
}

func hivelocityApiReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken string) {
	hivelocityApiProvisionOrReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken)
}

func hivelocityApiInfo() {
}
