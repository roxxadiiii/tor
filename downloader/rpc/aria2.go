package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Aria2Client struct {
	URL string
}

func New(url string) *Aria2Client {
	return &Aria2Client{URL: url}
}

type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type RPCResponse struct {
	ID     string      `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  interface{} `json:"error,omitempty"`
}

func (c *Aria2Client) Call(method string, params ...interface{}) (interface{}, error) {
	req := RPCRequest{
		Jsonrpc: "2.0",
		ID:      "1",
		Method:  method,
		Params:  params,
	}
	body, _ := json.Marshal(req)
	resp, err := http.Post(c.URL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	raw, _ := io.ReadAll(resp.Body)
	var rpcResp RPCResponse
	if err := json.Unmarshal(raw, &rpcResp); err != nil {
		return nil, err
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC Error: %v", rpcResp.Error)
	}
	return rpcResp.Result, nil
}
