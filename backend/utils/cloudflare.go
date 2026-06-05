package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ashbythorpe.com/website/config"
	"github.com/gofiber/fiber/v3/log"
)

type PurgeCacheRequestBody struct {
	Files []string `json:"files"`
}

func PurgeCloudflareCache(url string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	requestBody := PurgeCacheRequestBody{
		Files: []string{url},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Error(err)
		return
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/purge_cache", config.CloudflareZoneID),
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		log.Error(err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", config.CloudflareAPIToken))

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Error("Purging cache failed")
	}
}
