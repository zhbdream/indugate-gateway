# Python Integration Example

This directory will contain Python integration examples for connecting to InduGate via REST API and MCP.

```python
import requests

BASE_URL = "http://localhost:8080/api/v1"

# List devices
resp = requests.get(f"{BASE_URL}/devices")
print(resp.json())
```
