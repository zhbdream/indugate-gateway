package mcp

func toolDefinitions() []Tool {
	return []Tool{
		{
			Name:        "list_devices",
			Title:       "List Devices",
			Description: "List all configured industrial devices with their protocol, address, and connection status.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"protocol": map[string]any{
						"type":        "string",
						"description": "Optional filter by protocol: opcua, modbus, mqtt, s7",
						"enum":        []string{"opcua", "modbus", "mqtt", "s7", "bacnet"},
					},
					"status": map[string]any{
						"type":        "string",
						"description": "Optional filter by status: connected, disconnected, error",
						"enum":        []string{"connected", "disconnected", "error"},
					},
				},
			},
		},
		{
			Name:        "read_data",
			Title:       "Read Data Point",
			Description: "Read a data point from a connected device. Node ID format depends on protocol: OPC UA node ID, Modbus register (e.g. holding:0), or MQTT topic.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"device_id", "node_id"},
				"properties": map[string]any{
					"device_id": map[string]any{
						"type":        "integer",
						"description": "Device ID from list_devices",
					},
					"node_id": map[string]any{
						"type":        "string",
						"description": "Data point identifier (node ID, register, or MQTT topic)",
					},
				},
			},
		},
		{
			Name:        "write_data",
			Title:       "Write Data Point",
			Description: "Write a value to a writable data point on a connected device.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"device_id", "node_id", "value"},
				"properties": map[string]any{
					"device_id": map[string]any{
						"type":        "integer",
						"description": "Device ID from list_devices",
					},
					"node_id": map[string]any{
						"type":        "string",
						"description": "Data point identifier",
					},
					"value": map[string]any{
						"description": "Value to write (type depends on data point)",
					},
				},
			},
		},
		{
			Name:        "subscribe_data",
			Title:       "Subscribe Data Changes",
			Description: "Subscribe to data point changes on a connected device. Returns a subscription ID for polling events via REST API.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"device_id", "node_ids"},
				"properties": map[string]any{
					"device_id": map[string]any{
						"type":        "integer",
						"description": "Device ID from list_devices",
					},
					"node_ids": map[string]any{
						"type":        "array",
						"description": "List of data point identifiers to subscribe",
						"items": map[string]any{
							"type": "string",
						},
						"minItems": 1,
					},
					"interval_ms": map[string]any{
						"type":        "integer",
						"description": "Polling interval in milliseconds (default 1000, used for Modbus polling subscriptions)",
						"default":     1000,
					},
				},
			},
		},
		{
			Name:        "get_device_info",
			Title:       "Get Device Info",
			Description: "Get detailed information about a device including connection status and sample data points.",
			InputSchema: map[string]any{
				"type":     "object",
				"required": []string{"device_id"},
				"properties": map[string]any{
					"device_id": map[string]any{
						"type":        "integer",
						"description": "Device ID",
					},
				},
			},
		},
	}
}
