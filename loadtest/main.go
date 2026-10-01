package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultTotalRequests = 1_000_000
	defaultWorkers       = 500
	defaultURL           = "http://localhost:8080/messages/message-1"
)

type Result struct {
	latency time.Duration
	success bool
}

func main() {
	totalRequests := getEnvInt("TOTAL_REQUESTS", defaultTotalRequests)
	workers := getEnvInt("WORKERS", defaultWorkers)
	url := getEnvString("URL", defaultURL)

	fmt.Println("Starting load test")
	fmt.Println("------------------")
	fmt.Printf("URL:             %s\n", url)
	fmt.Printf("Total requests:  %d\n", totalRequests)
	fmt.Printf("Workers:         %d\n", workers)
	fmt.Println()

	transport := &http.Transport{
		MaxIdleConns:        workers,
		MaxIdleConnsPerHost: workers,
		MaxConnsPerHost:     workers,
		IdleConnTimeout:     30 * time.Second,

		DisableKeepAlives: false,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// Request IDs are generated through a channel so we don't
	// need to create one million goroutines.
	requests := make(chan int, workers*2)
	results := make(chan Result, workers*2)

	var wg sync.WaitGroup

	var success atomic.Int64
	var failed atomic.Int64

	start := time.Now()

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			for range requests {
				requestStart := time.Now()
				err := makeRequest(client, url)
				latency := time.Since(requestStart)

				if err != nil {
					failed.Add(1)

					results <- Result{
						latency: latency,
						success: false,
					}

					continue
				}

				success.Add(1)
				results <- Result{
					latency: latency,
					success: true,
				}
			}
		}(i)
	}

	var latencyWG sync.WaitGroup
	var latencies []time.Duration

	latencyWG.Go(func() {

		latencies = make([]time.Duration, 0, totalRequests)

		for result := range results {
			latencies = append(latencies, result.latency)
		}
	})

	// Producer.
	for i := range totalRequests {
		requests <- i
	}

	close(requests)
	wg.Wait()
	close(results)
	latencyWG.Wait()
	elapsed := time.Since(start)

	printResults(
		totalRequests,
		success.Load(),
		failed.Load(),
		elapsed,
		latencies,
	)
}

func makeRequest(client *http.Client, url string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	// Consume the response body so the underlying
	// TCP connection can be reused.
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func printResults(
	totalRequests int,
	success int64,
	failed int64,
	elapsed time.Duration,
	latencies []time.Duration,
) {
	requestsPerSecond := float64(totalRequests) / elapsed.Seconds()

	fmt.Println()
	fmt.Println("Load test finished")
	fmt.Println("------------------")

	fmt.Printf("Requests:        %d\n", totalRequests)
	fmt.Printf("Successful:      %d\n", success)
	fmt.Printf("Failed:          %d\n", failed)
	fmt.Printf("Total time:      %s\n", elapsed)
	fmt.Printf("Requests/sec:    %.2f\n", requestsPerSecond)

	if len(latencies) == 0 {
		return
	}

	sortDurations(latencies)

	fmt.Println()
	fmt.Println("Latency")
	fmt.Println("-------")

	fmt.Printf("Min:             %s\n", latencies[0])
	fmt.Printf("P50:             %s\n", percentile(latencies, 0.50))
	fmt.Printf("P95:             %s\n", percentile(latencies, 0.95))
	fmt.Printf("P99:             %s\n", percentile(latencies, 0.99))
	fmt.Printf("Max:             %s\n", latencies[len(latencies)-1])
}

func percentile(values []time.Duration, percentile float64) time.Duration {
	if len(values) == 0 {
		return 0
	}

	index := int(float64(len(values)-1) * percentile)

	return values[index]
}

func sortDurations(values []time.Duration) {
	quickSort(values, 0, len(values)-1)
}

func quickSort(values []time.Duration, low, high int) {
	if low >= high {
		return
	}

	pivot := values[(low+high)/2]

	i := low
	j := high

	for i <= j {
		for values[i] < pivot {
			i++
		}

		for values[j] > pivot {
			j--
		}

		if i <= j {
			values[i], values[j] = values[j], values[i]
			i++
			j--
		}
	}

	if low < j {
		quickSort(values, low, j)
	}

	if i < high {
		quickSort(values, i, high)
	}
}

func getEnvInt(name string, defaultValue int) int {
	value := os.Getenv(name)

	if value == "" {
		return defaultValue
	}

	result, err := strconv.Atoi(value)
	if err != nil || result <= 0 {
		return defaultValue
	}

	return result
}

func getEnvString(name string, defaultValue string) string {
	value := os.Getenv(name)

	if value == "" {
		return defaultValue
	}

	return value
}
