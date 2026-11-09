package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
)

type asset struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Owner  string `json:"owner"`
	Site   string `json:"site"`
	Status string `json:"status"`
}

type configuration struct {
	Endpoint  string `json:"endpoint"`
	APIKey    string `json:"api_key"`
	BatchSize int    `json:"batch_size"`
}

func readInventory(path string) ([]asset, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if strings.Join(header, ",") != "id,name,owner,site,status" {
		return nil, errors.New("expected CSV columns: id,name,owner,site,status")
	}
	items := make([]asset, 0)
	seen := make(map[string]bool)
	for line := 2; ; line++ {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", line, err)
		}
		for i := range row {
			row[i] = strings.TrimSpace(row[i])
		}
		if row[0] == "" || seen[row[0]] {
			return nil, fmt.Errorf("row %d: empty or duplicate asset ID %q", line, row[0])
		}
		seen[row[0]] = true
		items = append(items, asset{row[0], row[1], row[2], row[3], row[4]})
	}
	return items, nil
}

func run() error {
	input := flag.String("input", "inventory.csv", "inventory CSV path")
	configPath := flag.String("config", "config.json", "configuration path")
	dryRun := flag.Bool("dry-run", false, "print batches without sending")
	flag.Parse()
	data, err := os.ReadFile(*configPath)
	if err != nil {
		return err
	}
	var config configuration
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	if config.BatchSize < 1 {
		return errors.New("batch_size must be positive")
	}
	if config.Endpoint == "" || config.APIKey == "" {
		return errors.New("endpoint and api_key are required")
	}
	items, err := readInventory(*input)
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	for start := 0; start < len(items); start += config.BatchSize {
		end := min(start+config.BatchSize, len(items))
		batch := batchPayload{Assets: items[start:end]}
		if *dryRun {
			if err := encoder.Encode(batch); err != nil {
				return err
			}
		} else if err := sendBatch(config, batch); err != nil {
			return fmt.Errorf("batch %d: %w", start/config.BatchSize+1, err)
		}
	}
	if !*dryRun {
		fmt.Printf("Synced %d assets\n", len(items))
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
