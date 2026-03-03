package model

// Metadata represents the transparent context metadata collected at a node
type Metadata struct {
	ReqAll   map[string]string `json:"req_all"`
	ReqOnce  map[string]string `json:"req_once"`
	RespAll  map[string]string `json:"resp_all"`
	RespOnce map[string]string `json:"resp_once"`
}

// Node represents a service node in the call chain
type Node struct {
	Service  string   `json:"service"`
	Metadata Metadata `json:"metadata"`
}

// ChainResponse represents the response containing the call chain metadata
type ChainResponse struct {
	Message string `json:"message"`
	Chain   []Node `json:"chain"`
}
