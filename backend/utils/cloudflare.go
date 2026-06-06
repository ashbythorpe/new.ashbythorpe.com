package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"ashbythorpe.com/website/config"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

const purgeCacheServiceName = "Cloudflare Cache Purging Service"

// PurgeCacheService implements fiber.Service
type PurgeCacheService struct {
	queue  chan string
	state  string
	cancel context.CancelFunc
	done chan struct{}
}

func NewPurgeCacheService() *PurgeCacheService {
	return &PurgeCacheService{
		queue: make(chan string, 500),
		state: "initialized",
		done: make(chan struct{}),
	}
}

func (s *PurgeCacheService) Start(ctx context.Context) error {
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	go s.runWorker(workerCtx)

	s.state = "running"

	return nil
}

func (s *PurgeCacheService) runWorker(ctx context.Context) {
	defer close(s.done)

	batch := make(map[string]struct{})
	ticker := time.NewTicker(2 * time.Second)

	for {
		select {
		case url := <-s.queue:
			batch[url] = struct{}{}

			if len(batch) >= 30 {
				sendPurgeRequest(batch)
				batch = make(map[string]struct{})
				ticker.Reset(2 * time.Second)
			}

		case <-ticker.C:
			if len(batch) > 0 {
				sendPurgeRequest(batch)
				batch = make(map[string]struct{})
			}
		case <-ctx.Done():
			if len(batch) > 0 {
				sendPurgeRequest(batch)
			}
			return
		}
	}
}

func (s *PurgeCacheService) String() string {
	return purgeCacheServiceName
}

func (s *PurgeCacheService) State(ctx context.Context) (string, error) {
	return s.state, nil
}

func (s *PurgeCacheService) Terminate(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}

	select {
	case <-s.done:
	case <-ctx.Done():
		log.Error("Could not shut down service: ", ctx.Err())
		s.state = "terminated"
		return ctx.Err()
	}

	s.state = "terminated"
	return nil
}

func PurgeCloudflareCache(c fiber.Ctx, url string) {
	service := fiber.MustGetService[*PurgeCacheService](c.App().State(), purgeCacheServiceName)

	select {
	case service.queue <- url:
	default:
		log.Warn("Purge log full")
	}
}

type PurgeCacheRequestBody struct {
	Files []string `json:"files"`
}

func sendPurgeRequest(urlSet map[string]struct{}) {
	var urls []string
	for url := range urlSet {
		urls = append(urls, url)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	requestBody := PurgeCacheRequestBody{
		Files: urls,
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
