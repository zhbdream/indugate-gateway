package mcp

func promptDefinitions() []Prompt {
	return []Prompt{
		{
			Name:        "connect_device",
			Description: "Guide for connecting to an industrial device and reading its data points.",
			Arguments: []PromptArgument{
				{Name: "device_id", Description: "Device ID to connect", Required: true},
			},
		},
		{
			Name:        "monitor_telemetry",
			Description: "Guide for subscribing to and monitoring industrial telemetry data.",
			Arguments: []PromptArgument{
				{Name: "device_id", Description: "Device ID to monitor", Required: true},
				{Name: "node_ids", Description: "Comma-separated node IDs or topics", Required: true},
			},
		},
		{
			Name:        "protocol_overview",
			Description: "Overview of supported industrial protocols and node ID formats in InduGate.",
		},
	}
}

func buildPromptMessages(name string, args map[string]string) PromptsGetResult {
	switch name {
	case "connect_device":
		deviceID := args["device_id"]
		return PromptsGetResult{
			Description: "Connect to a device and read data",
			Messages: []PromptMessage{
				{
					Role: "user",
					Content: struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}{
						Type: "text",
						Text: "Help me connect to industrial device " + deviceID + " via InduGate and read its data points.",
					},
				},
				{
					Role: "assistant",
					Content: struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}{
						Type: "text",
						Text: "Steps:\n1. Call list_devices or get_device_info to verify device " + deviceID + "\n2. Ensure the device status is connected (use REST POST /api/v1/devices/" + deviceID + "/connect if needed)\n3. Browse nodes or use known node_id\n4. Call read_data with device_id and node_id",
					},
				},
			},
		}
	case "monitor_telemetry":
		return PromptsGetResult{
			Description: "Subscribe and monitor telemetry",
			Messages: []PromptMessage{
				{
					Role: "user",
					Content: struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}{
						Type: "text",
						Text: "Monitor device " + args["device_id"] + " nodes: " + args["node_ids"],
					},
				},
				{
					Role: "assistant",
					Content: struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}{
						Type: "text",
						Text: "Use subscribe_data with device_id and node_ids array, then poll subscription events via REST API.",
					},
				},
			},
		}
	default:
		return PromptsGetResult{
			Description: "InduGate protocol overview",
			Messages: []PromptMessage{
				{
					Role: "user",
					Content: struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}{Type: "text", Text: "What protocols does InduGate support?"},
				},
				{
					Role: "assistant",
					Content: struct {
						Type string `json:"type"`
						Text string `json:"text"`
					}{
						Type: "text",
						Text: serverInstructions,
					},
				},
			},
		}
	}
}
