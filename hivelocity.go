package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func hivelocityGetCloudInitScript(xnodeId string, xnodeAccessToken string) string {
	return "#cloud-config \nruncmd: \n - \"mkdir /tmp/boot && mount -t tmpfs -osize=90% none /tmp/boot && mkdir /tmp/boot/__img && wget -q -O /tmp/boot/__img/kexec.tar.xz http://boot.opnm.sh/kexec.tar.xz && mkdir /tmp/boot/system && mkdir /tmp/boot/system/proc && mount -t proc /proc /tmp/boot/system/proc && tar xvf /tmp/boot/__img/kexec.tar.xz -C /tmp/boot/system && rm /tmp/boot/__img/kexec.tar.xz && chroot /tmp/boot/system ./kexec_nixos \\\"-- XNODE_UUID=" + xnodeId + " XNODE_ACCESS_TOKEN=" + xnodeAccessToken + "\\\"\""

}

func hivelocityApiProvisionOrReset(hveApiKey, instanceId, xnodeId, xnodeAccessToken string) {

	// TODO: Make more robust, check region availability before provisioning.
	// Then cycle.

	body, err := json.Marshal(map[string]interface{}{
		"osName": "Ubuntu 22.04 (VPS)",
		"period": "monthly",
		// XXX: Might have to change region depending on settings.
		"locationName": "NYC1",
		// "productId": "2313", // Subject to future change (2311 for testing)

		// XXX: Change this to custom product id.
		"productId":   "2311",
		"hostname":    "xnode.openmesh.network",
		"forceReload": true,

		"script": hivelocityGetCloudInitScript(xnodeId, xnodeAccessToken),
	})

	req, err := http.NewRequest("POST", "https://core.hivelocity.net/api/v2/compute/"+instanceId, bytes.NewBuffer(body))

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

		fmt.Printf("res.Body: %v\n", val)
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
