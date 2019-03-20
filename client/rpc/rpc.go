package rpc

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/mit-dci/go-bverify/client"
)

type RpcServer struct {
	cli *client.Client
}

func NewRpcServer(c *client.Client) *RpcServer {
	return &RpcServer{cli: c}
}

type StartLogParameters struct {
	InitialStatement string
}

type StartLogReply struct {
	LogID string
}

func (s *RpcServer) StartLog(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var params StartLogParameters
	err := decoder.Decode(&params)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding json: %s", err.Error()))
		return
	}

	hexStatement, err := hex.DecodeString(params.InitialStatement)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding hex: %s", err.Error()))
		return
	}

	logId, err := s.cli.StartLog(hexStatement)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding json: %s", err.Error()))
		return
	}

	reply := StartLogReply{}
	reply.LogID = hex.EncodeToString(logId[:])
	json.NewEncoder(w).Encode(reply)
}

type AppendLogParameters struct {
	LogID     string
	Index     uint64
	Statement string
}

type AppendLogReply struct {
	Success bool
}

func (s *RpcServer) AppendLog(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var params AppendLogParameters
	err := decoder.Decode(&params)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding json: %s", err.Error()))
		return
	}

	hexLogID, err := hex.DecodeString(params.LogID)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding hex: %s", err.Error()))
		return
	}

	hexStatement, err := hex.DecodeString(params.Statement)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding hex: %s", err.Error()))
		return
	}

	logId32 := [32]byte{}
	copy(logId32[:], hexLogID)

	err = s.cli.AppendLog(params.Index, logId32, hexStatement)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error decoding json: %s", err.Error()))
		return
	}

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

func (s *RpcServer) Start() error {
	r := mux.NewRouter()

	r.HandleFunc("/start", s.StartLog)
	r.HandleFunc("/append", s.AppendLog)

	return http.ListenAndServe("localhost:8001", r)
}
