package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// deliverToInbox POSTs an AP activity to a single inbox URL.
func deliverToInbox(inboxURL string, activity any) error {
	body, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("marshal activity: %w", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", inboxURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/activity+json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POST %s: %w", inboxURL, err)
	}
	resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("POST %s: status %d", inboxURL, resp.StatusCode)
	}
	return nil
}

// deliverToPeers sends an activity to all known peer servers' inboxes.
func (h *Handler) deliverToPeers(activity any) {
	for _, peer := range h.cfg.BIKPeers {
		inboxURL := peer + "/bik/inbox"
		go func(url string) {
			if err := deliverToInbox(url, activity); err != nil {
				log.Printf("federation: deliver to %s: %v", url, err)
			} else {
				log.Printf("federation: delivered to %s", url)
			}
		}(inboxURL)
	}
}
