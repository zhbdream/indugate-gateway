package opcua

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/id"
	"github.com/gopcua/opcua/ua"
	"github.com/indugate/gateway/internal/protocol"
)

const defaultMaxDepth = 5

func (d *Driver) Browse(ctx context.Context, nodeIDStr string, maxDepth int) ([]protocol.NodeInfo, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}

	nodeID, err := ua.ParseNodeID(nodeIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid node id %q: %w", nodeIDStr, err)
	}

	return browseRecursive(ctx, d.client, d.client.Node(nodeID), "", 0, maxDepth)
}

func (d *Driver) BrowseChildren(ctx context.Context, nodeIDStr string) ([]protocol.NodeInfo, error) {
	if !d.IsConnected() {
		return nil, ErrNotConnected
	}

	nodeID, err := ua.ParseNodeID(nodeIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid node id %q: %w", nodeIDStr, err)
	}

	return browseOneLevel(ctx, d.client.Node(nodeID), "")
}

func browseRecursive(ctx context.Context, client *opcua.Client, n *opcua.Node, path string, level, maxDepth int) ([]protocol.NodeInfo, error) {
	if level > maxDepth {
		return nil, nil
	}

	nodes, err := browseOneLevel(ctx, n, path)
	if err != nil {
		return nil, err
	}

	var result []protocol.NodeInfo
	for _, info := range nodes {
		result = append(result, info)
		if info.HasChildren && level < maxDepth {
			childID, err := ua.ParseNodeID(info.NodeID)
			if err != nil {
				continue
			}
			children, err := browseRecursive(ctx, client, client.Node(childID), info.Path, level+1, maxDepth)
			if err != nil {
				return nil, err
			}
			result = append(result, children...)
		}
	}
	return result, nil
}

func browseOneLevel(ctx context.Context, n *opcua.Node, path string) ([]protocol.NodeInfo, error) {
	refTypes := []uint32{id.HierarchicalReferences, id.HasComponent, id.Organizes, id.HasProperty}
	seen := make(map[string]struct{})
	var nodes []protocol.NodeInfo

	for _, refType := range refTypes {
		refs, err := n.ReferencedNodes(ctx, refType, ua.BrowseDirectionForward, ua.NodeClassAll, true)
		if err != nil {
			continue
		}
		for _, ref := range refs {
			info, err := nodeInfoFromRef(ctx, ref, path)
			if err != nil || info == nil {
				continue
			}
			if _, ok := seen[info.NodeID]; ok {
				continue
			}
			seen[info.NodeID] = struct{}{}
			nodes = append(nodes, *info)
		}
	}

	return nodes, nil
}

func nodeInfoFromRef(ctx context.Context, n *opcua.Node, parentPath string) (*protocol.NodeInfo, error) {
	attrs, err := n.Attributes(
		ctx,
		ua.AttributeIDNodeClass,
		ua.AttributeIDBrowseName,
		ua.AttributeIDDisplayName,
		ua.AttributeIDDescription,
		ua.AttributeIDAccessLevel,
		ua.AttributeIDDataType,
	)
	if err != nil {
		return nil, err
	}

	info := &protocol.NodeInfo{NodeID: n.ID.String()}

	if attrs[0].Status == ua.StatusOK {
		info.NodeClass = nodeClassName(ua.NodeClass(attrs[0].Value.Int()))
	}
	if attrs[1].Status == ua.StatusOK {
		info.BrowseName = attrs[1].Value.String()
	}
	if attrs[2].Status == ua.StatusOK {
		info.DisplayName = attrs[2].Value.String()
	}
	if attrs[3].Status == ua.StatusOK {
		info.Description = attrs[3].Value.String()
	}
	if attrs[4].Status == ua.StatusOK {
		access := ua.AccessLevelType(attrs[4].Value.Int())
		info.Writable = access&ua.AccessLevelTypeCurrentWrite == ua.AccessLevelTypeCurrentWrite
	}
	if attrs[5].Status == ua.StatusOK {
		info.DataType = mapDataType(attrs[5].Value.NodeID())
	}

	info.Path = joinPath(parentPath, info.BrowseName)
	info.HasChildren = info.NodeClass == "Object" || info.NodeClass == "ObjectType" || info.NodeClass == "Folder"

	return info, nil
}

func mapDataType(nodeID *ua.NodeID) string {
	if nodeID == nil {
		return ""
	}
	switch v := nodeID.IntID(); v {
	case id.DateTime, id.UtcTime:
		return "time.Time"
	case id.Boolean:
		return "bool"
	case id.SByte:
		return "int8"
	case id.Int16:
		return "int16"
	case id.Int32:
		return "int32"
	case id.Byte:
		return "byte"
	case id.UInt16:
		return "uint16"
	case id.UInt32:
		return "uint32"
	case id.String:
		return "string"
	case id.Float:
		return "float32"
	case id.Double:
		return "float64"
	default:
		return nodeID.String()
	}
}

func nodeClassName(class ua.NodeClass) string {
	switch class {
	case ua.NodeClassObject:
		return "Object"
	case ua.NodeClassVariable:
		return "Variable"
	case ua.NodeClassMethod:
		return "Method"
	case ua.NodeClassObjectType:
		return "ObjectType"
	case ua.NodeClassVariableType:
		return "VariableType"
	case ua.NodeClassReferenceType:
		return "ReferenceType"
	case ua.NodeClassDataType:
		return "DataType"
	case ua.NodeClassView:
		return "View"
	default:
		return strconv.Itoa(int(class))
	}
}

func joinPath(parent, name string) string {
	if parent == "" {
		return name
	}
	if name == "" {
		return parent
	}
	return parent + "." + name
}
