package client

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mit-dci/go-bverify/crypto/fastsha256"
	"github.com/mit-dci/go-bverify/crypto/sig64"
	"github.com/mit-dci/go-bverify/logging"
	"github.com/mit-dci/go-bverify/wire"

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

type StatusReply struct {
	BlockHeight int32 `json:"blockHeight"`
	Synced      bool  `json:"synced"`
}

// Status is an RPC method to fetch the status of the server
func (s *RpcServer) Status(w http.ResponseWriter, r *http.Request) {
	reply := StatusReply{}
	reply.BlockHeight = s.cli.SPVHeight()
	reply.Synced = s.cli.SPVSynced()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(reply)
}

// AddForeignLog is an RPC method to instruct this client to keep updated
// proofs for the given log statement
func (s *RpcServer) AddForeignLog(w http.ResponseWriter, r *http.Request) {
	// The passed in bytes should be deserializable as a ForeignStatement
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.writeError(w, fmt.Errorf("Could not read proof from request body: %s", err.Error()))
		return
	}

	dec, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		s.writeError(w, fmt.Errorf("Request body is not valid base64: %s", err.Error()))
		return
	}

	fs := wire.ForeignStatementFromBytes(dec)

	err = s.cli.AddForeignLog(fs)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error adding foreign log: %s", err.Error()))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(true)
}

type VerificationResult struct {
	Statement      string
	Valid          bool
	Error          string
	PubKey         string
	BlockHash      string
	TxHash         string
	BlockTimestamp int64
}

func (s *RpcServer) VerifyOnce(w http.ResponseWriter, r *http.Request) {
	verify := func() VerificationResult {
		v := VerificationResult{}

		// The passed in bytes should be deserializable as a ForeignStatement
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			v.Valid = false
			v.Error = fmt.Sprintf("Could not read proof from request body: %s", err.Error())
			return v
		}

		dec, err := base64.StdEncoding.DecodeString(string(b))
		if err != nil {
			v.Valid = false
			v.Error = fmt.Sprintf("Request body is not valid base64: %s", err.Error())
			return v
		}

		fs := wire.ForeignStatementFromBytes(dec)
		logId, hash, err := s.cli.GetForeignLogIDAndHash(fs)
		if err != nil {
			v.Valid = false
			v.Error = fmt.Sprintf("Could not read foreign statement: %s", err.Error())
			return v
		}

		// TODO: Get proof from server if it's nil?

		val, err := fs.Proof.Get(logId[:])
		if err != nil || !bytes.Equal(val, hash[:]) {
			v.Valid = false
			v.Error = fmt.Sprintf("Provided proof does not contain correct value for the provided log: %v - [%x vs %x]", err, val, hash)
			return v
		}

		// Calculate the commitment from the partial tree we got from the
		// server and check if it is a known commitment
		rootHash := fs.Proof.Commitment()
		c, err := s.cli.getCommitment(rootHash)
		if err != nil {
			v.Valid = false
			v.Error = fmt.Sprintf("Could not get commitment from the given proof: %s", err.Error())
			return v
		}

		block, err := s.cli.GetBlockHeaderByHash(c.IncludedInBlock)
		if err != nil {
			v.Valid = false
			v.Error = "Could not fetch block details for this log's last committed statement"
			return v
		}

		v.Valid = true
		v.BlockHash = c.IncludedInBlock.String()
		v.PubKey = hex.EncodeToString(fs.PubKey[:])
		v.Statement = fs.StatementPreimage
		v.TxHash = c.TxHash.String()
		v.BlockTimestamp = block.Timestamp.Unix()
		return v
	}

	result := verify()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(result)

}

type LogInfo struct {
	Foreign                 bool
	LogID                   string
	LastStatement           string
	LastIndex               int64
	LastCommitment          string
	LastCommitmentProof     string
	LastCommitmentBlock     string
	LastCommitmentTimestamp int64
	Valid                   bool
	Error                   string
}

// Logs will return a list of known logs and their last statements and proofs
func (s *RpcServer) Logs(w http.ResponseWriter, r *http.Request) {
	logIds, err := s.cli.GetAllLogIDs()
	if err != nil {
		s.writeError(w, fmt.Errorf("Unable to fetch logs: %s", err.Error()))
		return
	}

	logs := make([]LogInfo, len(logIds))
	for i, l := range logIds {
		logs[i].Foreign = s.cli.IsForeignLog(l)
		logs[i].LogID = hex.EncodeToString(l[:])

		idx, _, err := s.cli.GetLastCommittedLog(l)
		if err == nil {
			logs[i].Valid = true
			logs[i].LastIndex = idx
			logs[i].LastStatement, err = s.cli.GetLogPreimage(l, uint64(idx))
			if err != nil {
				logs[i].Valid = false
				logs[i].Error = "Could not fetch preimage for this log's last committed statement"

			} else {
				lastCommitHex, err := s.cli.GetLogCommitment(l, uint64(idx))
				if err != nil {
					logs[i].Valid = false
					logs[i].Error = "Could not fetch the commitment hash for this log's last committed statement"
				} else {
					logs[i].LastCommitment = hex.EncodeToString(lastCommitHex[:])

					comm, err := s.cli.getCommitment(lastCommitHex[:])
					if err != nil {
						logs[i].Valid = false
						logs[i].Error = "Could not fetch commitment details for this log's last committed statement"
					} else {
						logs[i].LastCommitmentProof = hex.EncodeToString(comm.MerkleProof.Bytes())
						logs[i].LastCommitmentBlock = hex.EncodeToString(comm.IncludedInBlock[:])
						block, err := s.cli.GetBlockHeaderByHash(comm.IncludedInBlock)
						if err != nil {
							logs[i].Valid = false
							logs[i].Error = "Could not fetch block details for this log's last committed statement"
						} else {
							logs[i].LastCommitmentTimestamp = block.Timestamp.Unix()
						}
					}

				}
			}
		} else {
			logs[i].Valid = false
			logs[i].Error = "Could not find a commitment for this log (yet)"
		}

	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(logs)
}

// Logs will return a list of known logs and their last statements and proofs
func (s *RpcServer) Export(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	logIdHex, err := hex.DecodeString(vars["logId"])
	if err != nil || len(logIdHex) != 32 {
		s.writeError(w, fmt.Errorf("Invalid Log ID"))
		return
	}
	logId32 := [32]byte{}
	copy(logId32[:], logIdHex)

	idx, _, err := s.cli.GetLastCommittedLog(logId32)
	if err != nil {
		s.writeError(w, fmt.Errorf("Error fetching last committed log: %s", err.Error()))
		return
	}

	if s.cli.IsForeignLog(logId32) {
		s.writeError(w, fmt.Errorf("Cannot export a foreign log. The original sender should export it."))
		return
	}

	fs := &wire.ForeignStatement{}
	fs.Index = uint64(idx)
	fs.InitialStatement = (idx == 0)
	fs.LogID = logId32
	fs.StatementPreimage, err = s.cli.GetLogPreimage(logId32, uint64(idx))
	if err != nil {
		s.writeError(w, fmt.Errorf("Error fetching last committed statement: %s", err.Error()))
		return
	}
	fs.PubKey = s.cli.pubKey

	commitment, err := s.cli.GetLogCommitment(logId32, uint64(idx))
	if err != nil {
		s.writeError(w, fmt.Errorf("Error fetching commitment hash for last committed statement: %s", err.Error()))
		return
	}

	proof, err := s.cli.GetProofForCommitment(commitment, [][]byte{logIdHex})
	if err != nil {
		s.writeError(w, fmt.Errorf("Error fetching commitment hash for last committed statement: %s", err.Error()))
		return
	}

	fs.Proof = proof

	// Recreate signature. Maybe this is somewhat ugly, but... figure it out later
	statementHash := fastsha256.Sum256([]byte(fs.StatementPreimage))
	signaturePayload := []byte{}
	if fs.InitialStatement {
		signaturePayload = wire.NewSignedCreateLogStatement(fs.PubKey, statementHash[:]).CreateStatement.Bytes()
	} else {
		signaturePayload = wire.NewSignedLogStatement(uint64(fs.Index), logId32, statementHash[:]).Statement.Bytes()
	}
	signatureHash := fastsha256.Sum256(signaturePayload)
	sig, err := s.cli.key.Sign(signatureHash[:])
	if err != nil {
		s.writeError(w, fmt.Errorf("Could not recreate signature: %s", err.Error()))
		return
	}

	csig, err := sig64.SigCompress(sig.Serialize())
	if err != nil {
		s.writeError(w, fmt.Errorf("Could not compress signature: %s", err.Error()))
		return
	}

	fs.Signature = csig

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(fs.Bytes())
}

type JsonCommitment struct {
	Commitment             string
	TxHash                 string
	TriggeredAtBlockHeight int
	IncludedInBlock        string
}

// Commitments will return a list of server commitments
func (s *RpcServer) Commitments(w http.ResponseWriter, r *http.Request) {
	comms, err := s.cli.getAllCommitments()
	if err != nil {
		s.writeError(w, fmt.Errorf("Error reading commitments: %s", err.Error()))
		return
	}

	jsonComms := make([]JsonCommitment, len(comms))
	for i, c := range comms {
		jsonComms[i] = JsonCommitment{
			Commitment:             hex.EncodeToString(c.Commitment[:]),
			IncludedInBlock:        hex.EncodeToString(c.IncludedInBlock[:]),
			TxHash:                 hex.EncodeToString(c.TxHash[:]),
			TriggeredAtBlockHeight: c.TriggeredAtBlockHeight,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(jsonComms)
}

// Start starts the RPC server listening to client connections
func (s *RpcServer) Start() error {
	r := mux.NewRouter()

	r.HandleFunc("/start", s.StartLog).Methods("POST")
	r.HandleFunc("/append", s.AppendLog).Methods("POST")
	r.HandleFunc("/status", s.Status).Methods("GET")
	r.HandleFunc("/addforeignlog", s.AddForeignLog).Methods("POST")
	r.HandleFunc("/verifyonce", s.VerifyOnce).Methods("POST")
	r.HandleFunc("/logs", s.Logs).Methods("GET")
	r.HandleFunc("/commitments", s.Commitments).Methods("GET")
	r.HandleFunc("/export/{logId}", s.Export).Methods("GET")

	logging.Debugf("Server is listening on localhost:8001")

	return http.ListenAndServe("localhost:8001", r)
}
