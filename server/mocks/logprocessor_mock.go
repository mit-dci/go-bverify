// Code generated by MockGen. DO NOT EDIT.
// Source: logprocessor.go

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	mpt "github.com/mit-dci/go-bverify/mpt"
	wire "github.com/mit-dci/go-bverify/wire"
)

// MockLogProcessor is a mock of LogProcessor interface
type MockLogProcessor struct {
	ctrl     *gomock.Controller
	recorder *MockLogProcessorMockRecorder
}

// MockLogProcessorMockRecorder is the mock recorder for MockLogProcessor
type MockLogProcessorMockRecorder struct {
	mock *MockLogProcessor
}

// NewMockLogProcessor creates a new mock instance
func NewMockLogProcessor(ctrl *gomock.Controller) *MockLogProcessor {
	mock := &MockLogProcessor{ctrl: ctrl}
	mock.recorder = &MockLogProcessorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockLogProcessor) EXPECT() *MockLogProcessorMockRecorder {
	return m.recorder
}

// Process mocks base method
func (m *MockLogProcessor) Process() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Process")
}

// Process indicates an expected call of Process
func (mr *MockLogProcessorMockRecorder) Process() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Process", reflect.TypeOf((*MockLogProcessor)(nil).Process))
}

// SendProofs mocks base method
func (m *MockLogProcessor) SendProofs(delta *mpt.DeltaMPT) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendProofs", delta)
	ret0, _ := ret[0].(error)
	return ret0
}

// SendProofs mocks base method
func (m *MockLogProcessor) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// SendProofs indicates an expected call of SendProofs
func (mr *MockLogProcessorMockRecorder) SendProofs(delta interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendProofs", reflect.TypeOf((*MockLogProcessor)(nil).SendProofs), delta)
}

// ProcessMessage mocks base method
func (m_2 *MockLogProcessor) ProcessMessage(t wire.MessageType, m []byte) error {
	m_2.ctrl.T.Helper()
	ret := m_2.ctrl.Call(m_2, "ProcessMessage", t, m)
	ret0, _ := ret[0].(error)
	return ret0
}

// ProcessMessage indicates an expected call of ProcessMessage
func (mr *MockLogProcessorMockRecorder) ProcessMessage(t, m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessMessage", reflect.TypeOf((*MockLogProcessor)(nil).ProcessMessage), t, m)
}

// ProcessMessage indicates an expected call of ProcessMessage
func (mr *MockLogProcessorMockRecorder) Stop(t, m interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessMessage", reflect.TypeOf((*MockLogProcessor)(nil).ProcessMessage), t, m)
}

// ProcessRequestProof mocks base method
func (m *MockLogProcessor) ProcessRequestProof(msg *wire.RequestProofMessage) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProcessRequestProof", msg)
	ret0, _ := ret[0].(error)
	return ret0
}

// ProcessRequestProof indicates an expected call of ProcessRequestProof
func (mr *MockLogProcessorMockRecorder) ProcessRequestProof(msg interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessRequestProof", reflect.TypeOf((*MockLogProcessor)(nil).ProcessRequestProof), msg)
}

// ProcessCreateLog mocks base method
func (m *MockLogProcessor) ProcessCreateLog(scls *wire.SignedCreateLogStatement) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProcessCreateLog", scls)
	ret0, _ := ret[0].(error)
	return ret0
}

// ProcessCreateLog indicates an expected call of ProcessCreateLog
func (mr *MockLogProcessorMockRecorder) ProcessCreateLog(scls interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessCreateLog", reflect.TypeOf((*MockLogProcessor)(nil).ProcessCreateLog), scls)
}

// ProcessAppendLog mocks base method
func (m *MockLogProcessor) ProcessAppendLog(sls *wire.SignedLogStatement) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ProcessAppendLog", sls)
	ret0, _ := ret[0].(error)
	return ret0
}

// ProcessAppendLog indicates an expected call of ProcessAppendLog
func (mr *MockLogProcessorMockRecorder) ProcessAppendLog(sls interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ProcessAppendLog", reflect.TypeOf((*MockLogProcessor)(nil).ProcessAppendLog), sls)
}
