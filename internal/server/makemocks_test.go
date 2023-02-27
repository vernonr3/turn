package server

import (
	"io"
	"net"
	"testing"

	"github.com/pion/stun"
)

func makeMockNetAddr(t *testing.T, addressString string) *mockUDPNetAddrImpl {
	mMockNetAddrImpl := NewUDPMockNetAddr()
	mMockNetAddrImpl.IP = net.IPv4(byte(127), byte(0), byte(0), byte(1))
	mMockNetAddrImpl.Port = 5434
	mMockNetAddrImpl.Zone = "link-local"

	mMockNetAddrImpl.handleString = func() string {
		return addressString
	}
	mMockNetAddrImpl.handleNetwork = func() string {
		return "udp"
	}
	return mMockNetAddrImpl
}

type AttrType uint16

type (
	// Setter sets *Message attribute.
	mockSetterIF interface {
		AddTo(m *stun.StunMessageIF) error
	}
	// Getter parses attribute from *Message.
	mockGetterIF interface {
		GetFrom(m *stun.StunMessageIF) error
	}
	// Checker checks *Message attribute.
	mockCheckerIF interface {
		Check(m *stun.StunMessageIF) error
	}
)

type stunvarsIF interface {
	GetTransactionID() [stun.TransactionIDSize]byte
	GetMessage() *stun.Message
	SetMessage(msg []byte) bool
}

type accessMessageType interface {
	GetTypeMethod() stun.Method
	GetTypeClass() stun.MessageClass
}

type mockStunMessageIF interface {
	stunvarsIF
	accessMessageType
	ApplyBuf(buf []byte)

	Build(setters ...stun.Setter) error
	Check(checkers ...stun.Checker)
	Parse(getters ...stun.Getter) error
	ForEach(t stun.AttrType, f func(m *stun.Message) error) error

	UnmarshalBinary(data []byte) error
	GobEncode() ([]byte, error)
	GobDecode(data []byte) error
	AddTo(b *stun.StunMessageIF) error
	NewTransactionID() error
	String() string
	Reset()
	Grow(n int)
	Add(t stun.AttrType, v []byte)
	Equal(b *stun.StunMessageIF) bool
	WriteLength()
	WriteHeader()
	WriteTransactionID()
	WriteAttributes()
	WriteType()
	SetType(t stun.MessageType)
	Encode()
	Decode() error
	WriteTo(w io.Writer) (int64, error)
	ReadFrom(r io.Reader) (int64, error)
	Write(tBuf []byte) (int, error)
	CloneTo(b *stun.StunMessageIF) error
	Contains(t stun.AttrType) bool
}

type mockStunMessageImpl struct {
	Type          stun.MessageType
	Length        uint32 // len(Raw) not including header
	TransactionID [stun.TransactionIDSize]byte
	Attributes    stun.Attributes
	Raw           []byte

	handleApplyBuf func(buf []byte)
	handleBuild    func(setters ...stun.Setter) error
	handleCheck    func(checkers ...stun.Checker)
	handleParse    func(getters ...stun.Getter) error
	handleForEach  func(t stun.AttrType, f func(m *stun.Message) error) error

	handleGetTypeMethod func() stun.Method
	handleGetTypeClass  func() stun.MessageClass

	handleGetTransactionID   func() [stun.TransactionIDSize]byte
	handleUnmarshalBinary    func(data []byte) error
	handleGobEncode          func() ([]byte, error)
	handleGobDecode          func(data []byte) error
	handleAddTo              func(b *stun.StunMessageIF) error
	handleNewTransactionID   func() error
	handleString             func() string
	handleReset              func()
	handleGrow               func(n int)
	handleAdd                func(t stun.AttrType, v []byte)
	handleEqual              func(b *stun.StunMessageIF) bool
	handleWriteLength        func()
	handleWriteHeader        func()
	handleWriteTransactionID func()
	handleWriteAttributes    func()
	handleWriteType          func()
	handleSetType            func(t stun.MessageType)
	handleEncode             func()
	handleDecode             func() error
	handleWriteTo            func(w io.Writer) (int64, error)
	handleReadFrom           func(r io.Reader) (int64, error)
	handleWrite              func(tBuf []byte) (int, error)
	handleCloneTo            func(b *stun.StunMessageIF) error
	handleContains           func(t stun.AttrType) bool
	handleGetMessage         func() *stun.Message
	handleAuth               func(username string, realm string, srcAddr net.Addr) (key []byte, ok bool)
}

func NewMockStunImplMessage() *mockStunMessageImpl {
	mockStunMessage := mockStunMessageImpl{}
	return &mockStunMessage
}

type mockStunIF interface {
	NewType()
}

type mockStunImpl struct {
	Message mockStunMessageImpl
}

func NewMockStunImpl() *mockStunImpl {
	mockStunMessage := mockStunMessageImpl{}
	mockStun := &mockStunImpl{
		Message: mockStunMessage,
	}
	return mockStun
}

func (m *mockStunImpl) NewType() {

}

func (m *mockStunMessageImpl) GetTransactionID() [stun.TransactionIDSize]byte {
	return m.handleGetTransactionID()
}

func (m *mockStunMessageImpl) GetMessage() *stun.Message {
	return m.handleGetMessage()
}

func (m *mockStunMessageImpl) SetMessage(msg []byte) bool {
	m.Raw = msg
	return true
}

func (m *mockStunMessageImpl) GetTypeMethod() stun.Method {
	return m.handleGetTypeMethod()
}
func (m *mockStunMessageImpl) GetTypeClass() stun.MessageClass {
	return m.handleGetTypeClass()
}

func (m *mockStunMessageImpl) ApplyBuf(buf []byte) {

}

func (m *mockStunMessageImpl) Build(setters ...stun.Setter) error {
	return m.handleBuild(setters...)
}
func (m *mockStunMessageImpl) Check(checkers ...stun.Checker) {
}
func (m *mockStunMessageImpl) Parse(getters ...stun.Getter) error {
	return m.handleParse(getters...)
}
func (m *mockStunMessageImpl) ForEach(t stun.AttrType, f func(m *stun.Message) error) error {
	return m.handleForEach(t, f)
}

func (m *mockStunMessageImpl) UnmarshalBinary(data []byte) error {
	return m.handleUnmarshalBinary(data)
}
func (m *mockStunMessageImpl) GobEncode() ([]byte, error) {
	return m.handleGobEncode()
}
func (m *mockStunMessageImpl) GobDecode(data []byte) error {
	return m.handleGobDecode(data)
}
func (m *mockStunMessageImpl) AddTo(b *stun.StunMessageIF) error {
	return m.handleAddTo(b)
}
func (m *mockStunMessageImpl) NewTransactionID() error {
	return m.handleNewTransactionID()
}
func (m *mockStunMessageImpl) String() string {
	return m.handleString()
}
func (m *mockStunMessageImpl) Reset() {
}
func (m *mockStunMessageImpl) Grow(n int) {
}
func (m *mockStunMessageImpl) Add(t stun.AttrType, v []byte) {
}
func (m *mockStunMessageImpl) Equal(b *stun.StunMessageIF) bool {
	return m.handleEqual(b)
}
func (m *mockStunMessageImpl) WriteLength() {

}
func (m *mockStunMessageImpl) WriteHeader() {

}
func (m *mockStunMessageImpl) WriteTransactionID() {

}
func (m *mockStunMessageImpl) WriteAttributes() {

}
func (m *mockStunMessageImpl) WriteType() {

}
func (m *mockStunMessageImpl) SetType(t stun.MessageType) {

}
func (m *mockStunMessageImpl) Encode() {

}

func (m *mockStunMessageImpl) Decode() error {
	return m.handleDecode()
}

func (m *mockStunMessageImpl) WriteTo(w io.Writer) (int64, error) {
	return m.handleWriteTo(w)
}
func (m *mockStunMessageImpl) ReadFrom(r io.Reader) (int64, error) {
	return m.handleReadFrom(r)
}
func (m *mockStunMessageImpl) Write(tBuf []byte) (int, error) {
	return m.handleWrite(tBuf)
}
func (m *mockStunMessageImpl) CloneTo(b *stun.StunMessageIF) error {
	return m.handleCloneTo(b)
}
func (m *mockStunMessageImpl) Contains(t stun.AttrType) bool {
	return m.handleContains(t)
}
