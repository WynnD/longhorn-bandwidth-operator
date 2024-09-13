package main

import (
	"fmt"
	"os"

	"sigs.k8s.io/yaml"
)

// NodeBandwidthConfig defines the bandwidth configuration for a node
type NodeBandwidthConfig struct {
	IngressLimit string `json:"ingress_limit"`
	EgressLimit  string `json:"egress_limit"`
}

// BandwidthConfig is the top-level configuration
type BandwidthConfig struct {
	Nodes map[string]NodeBandwidthConfig `json:"nodes"`
}

func loadConfig(path string) (BandwidthConfig, error) {
	var config BandwidthConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
