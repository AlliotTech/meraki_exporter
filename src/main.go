package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MerakiAPIResponse represents the structure of the response from Meraki API
type MerakiAPIResponse []struct {
	NetworkID string `json:"networkId"`
	Name      string `json:"name"`
	ByUplink  []struct {
		Serial    string `json:"serial"`
		Interface string `json:"interface"`
		Sent      int64  `json:"sent"`
		Received  int64  `json:"received"`
	} `json:"byUplink"`
}

// Define Prometheus metrics
var (
	sentBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "meraki_sent_bytes",
			Help: "Number of bytes sent over the uplink",
		},
		[]string{"networkId", "name", "serial", "interface"},
	)

	receivedBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "meraki_received_bytes",
			Help: "Number of bytes received over the uplink",
		},
		[]string{"networkId", "name", "serial", "interface"},
	)
)

var once sync.Once

func registerMetrics() {
	once.Do(func() {
		// Register metrics with Prometheus
		prometheus.MustRegister(sentBytes)
		prometheus.MustRegister(receivedBytes)
	})
}

var httpClient = &http.Client{
	Timeout: 30 * time.Second,
}

func fetchMerakiData(ctx context.Context, organizationID, apiKey, interval string) (MerakiAPIResponse, error) {
	url := fmt.Sprintf("https://api.meraki.com/api/v1/organizations/%s/appliance/uplinks/usage/byNetwork?timespan=%s", organizationID, interval)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}

	var data MerakiAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return data, nil
}

func updateMetrics(data MerakiAPIResponse) {
	for _, network := range data {
		for _, uplink := range network.ByUplink {
			labels := prometheus.Labels{
				"networkId": network.NetworkID,
				"name":      network.Name,
				"serial":    uplink.Serial,
				"interface": uplink.Interface,
			}
			sentBytes.With(labels).Set(float64(uplink.Sent))
			receivedBytes.With(labels).Set(float64(uplink.Received))
		}
	}
}

func main() {
	organizationID := os.Getenv("MERAKI_ORGANIZATION_ID")
	apiKey := os.Getenv("MERAKI_API_KEY")
	intervalStr := os.Getenv("INTERVAL")

	if organizationID == "" || apiKey == "" {
		log.Fatalf("MERAKI_ORGANIZATION_ID and MERAKI_API_KEY environment variables are required")
	}

	interval := 60 * time.Second // Default interval to 60 seconds
	if intervalStr != "" {
		if i, err := strconv.Atoi(intervalStr); err == nil {
			interval = time.Duration(i) * time.Second
		} else {
			log.Fatalf("Invalid INTERVAL value: %s", intervalStr)
		}
	}
	log.Printf("Running with Interval: %s", interval)

	registerMetrics()

	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), httpClient.Timeout)
			defer cancel()

			data, err := fetchMerakiData(ctx, organizationID, apiKey, strconv.Itoa(int(interval.Seconds())))
			if err != nil {
				log.Printf("Error fetching Meraki data: %v", err)
			} else {
				updateMetrics(data)
			}
			time.Sleep(interval)
		}
	}()
	http.Handle("/", http.RedirectHandler("/metrics", http.StatusMovedPermanently))
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}