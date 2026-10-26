package utils

import (
	"fmt"
	"net/http"
)

func HealthCheck() error {
	resp, err := http.Get("http://127.0.0.1:3000/health")

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health endpoint returned %s", resp.Status)
	}

	return nil
}
