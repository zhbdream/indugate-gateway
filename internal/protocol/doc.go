// Package protocol contains industrial protocol drivers used by InduGate.
//
// Each subpackage (opcua, modbus, mqtt, s7, bacnet) implements connect,
// browse, read, write, and subscribe for one protocol. Drivers are invoked
// through service.DriverManager, not directly from HTTP or MCP handlers.
package protocol
