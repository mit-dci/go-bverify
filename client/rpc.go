package client

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// RpcServer runs a simple JSON based API for clients to start and maintain
// logs using the client
type RpcServer struct {
	cli *Client
}

// NewRpcServer creates a new RPC server
func NewRpcServer(c *Client) *RpcServer {
	return &RpcServer{cli: c}
}

type StartLogParameters struct {
	InitialStatement string // The statement (in clear text) you wish to start the log with
}

type StartLogReply struct {
	LogID string // The Log ID (hex encoded bytes)
}

// StartLog is an RPC method to start a new log
func (s *RpcServer) StartLog(w http.ResponseWriter, r *http.Request) {
	// Decode the passed in parameters
	decoder := json.NewDecoder(r.Body)
	var params StartLogParameters
	err := decoder.Decode(&params)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding json: %s", err.Error()))
		return
	}

	// Start the log
	logId, err := s.cli.StartLogText(params.InitialStatement)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding json: %s", err.Error()))
		return
	}

	// Return the LogID as reply to the caller
	reply := StartLogReply{}
	reply.LogID = hex.EncodeToString(logId[:])
	json.NewEncoder(w).Encode(reply)
}

type AppendLogParameters struct {
	LogID     string // The ID of the log to append to (in hexadecimal format)
	Index     uint64 // The 0-based index of the statement that is to be written
	Statement string // The statement (in clear text) to append to the log
}

type AppendLogReply struct {
	Success bool
}

// AppendLog is an RPC method to append to an existing log
func (s *RpcServer) AppendLog(w http.ResponseWriter, r *http.Request) {
	// Decode the passed in parameters
	decoder := json.NewDecoder(r.Body)
	var params AppendLogParameters
	err := decoder.Decode(&params)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding json: %s", err.Error()))
		return
	}

	// Decode the passed in LogID
	hexLogID, err := hex.DecodeString(params.LogID)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding hex: %s", err.Error()))
		return
	}
	logId32 := [32]byte{}
	copy(logId32[:], hexLogID)

	// Append to the log
	err = s.cli.AppendLogText(params.Index, logId32, params.Statement)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding json: %s", err.Error()))
		return
	}

	// Generate a reply and send it back to the caller
	reply := AppendLogReply{}
	reply.Success = true
	json.NewEncoder(w).Encode(reply)
}

type ErrorResponse struct {
	Error        bool
	ErrorDetails string
}

func (s *RpcServer) writeError(w http.ResponseWriter, err error) {
	resp := ErrorResponse{Error: true, ErrorDetails: err.Error()}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(500)
	json.NewEncoder(w).Encode(resp)
}

// Start starts the RPC server listening to client connections
func (s *RpcServer) Start() error {
	r := mux.NewRouter()

	r.HandleFunc("/start", s.StartLog)
	r.HandleFunc("/append", s.AppendLog)

	return http.ListenAndServe("localhost:8001", r)
}
