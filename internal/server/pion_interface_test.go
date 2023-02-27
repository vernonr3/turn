package server

import (
	"fmt"
	"net"
	"net/netip"
	"time"
)

type mockLeveledLoggerIF interface {
	Trace(msg string)
	Tracef(format string, args ...interface{})
	Debug(msg string)
	Debugf(format string, args ...interface{})
	Info(msg string)
	Infof(format string, args ...interface{})
	Warn(msg string)
	Warnf(format string, args ...interface{})
	Error(msg string)
	Errorf(format string, args ...interface{})
}

type mockLeveledLoggerImpl struct {
}

func (m *mockLeveledLoggerImpl) Trace(msg string) {
	fmt.Println(msg)
}
func (m *mockLeveledLoggerImpl) Tracef(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
func (m *mockLeveledLoggerImpl) Debug(msg string) {
	fmt.Println(msg)
}
func (m *mockLeveledLoggerImpl) Debugf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
func (m *mockLeveledLoggerImpl) Info(msg string) {
	fmt.Println(msg)
}
func (m *mockLeveledLoggerImpl) Infof(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
func (m *mockLeveledLoggerImpl) Warn(msg string) {
	fmt.Println(msg)
}
func (m *mockLeveledLoggerImpl) Warnf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}
func (m *mockLeveledLoggerImpl) Error(msg string) {
	fmt.Println(msg)
}
func (m *mockLeveledLoggerImpl) Errorf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

func NewLeveledLogger() *mockLeveledLoggerImpl {
	mLogger := mockLeveledLoggerImpl{}
	return &mLogger
}

type mockPacketConnIF interface {
	LocalAddr() net.Addr
	ReadFrom(p []byte) (n int, addr net.Addr, err error)
	Close() error
	WriteTo(p []byte, addr net.Addr) (n int, err error)
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

type mockPacketConnImpl struct {
	mMockNewAddr    *mockNetAddrImpl
	handleLocalAddr func() net.Addr
	handleWriteTo   func(p []byte, addr net.Addr) (n int, err error)
	handleAuth      func(username string, realm string, srcAddr net.Addr) (key []byte, ok bool)
}

func (m *mockPacketConnImpl) LocalAddr() net.Addr {
	return m.handleLocalAddr()
}
func (m *mockPacketConnImpl) Close() error {
	return nil
}
func (m *mockPacketConnImpl) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	mMockNewAddr := NewMockNetAddr()
	return 0, mMockNewAddr, nil
}

func (m *mockPacketConnImpl) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	return m.handleWriteTo(p, addr)

}
func (m *mockPacketConnImpl) SetDeadline(t time.Time) error {
	return nil
}
func (m *mockPacketConnImpl) SetReadDeadline(t time.Time) error {
	return nil
}
func (m *mockPacketConnImpl) SetWriteDeadline(t time.Time) error {
	return nil
}

func NewMockPacketConnImpl() *mockPacketConnImpl {
	mMock := mockPacketConnImpl{}
	mMock.mMockNewAddr = NewMockNetAddr()
	return &mMock
}

type mockNetAddrIF interface {
	Network() string
	String() string
}
type mockNetAddrImpl struct {
	net.UDPAddr
	handleString  func() string
	handleNetwork func() string
}

func NewMockNetAddr() *mockNetAddrImpl {
	mNetAddrImpl := mockNetAddrImpl{}
	return &mNetAddrImpl
}

func (m *mockNetAddrImpl) String() string {
	return m.handleString()
}

func (m *mockNetAddrImpl) Network() string {
	return m.handleNetwork()
}

type mockUDPNetAddrIF interface {
	Network() string
	String() string
	AddrPort() netip.AddrPort
	ResolveUDPAddr(network, address string) (*net.UDPAddr, error)
	UDPAddrFromAddrPort(addr netip.AddrPort) *net.UDPAddr
}
type mockUDPNetAddrImpl struct {
	net.UDPAddr
	/*	IP                        net.IP
		Port                      int
		Zone                      string*/
	handleString              func() string
	handleNetwork             func() string
	handleAddrPort            func() netip.AddrPort
	handleResolveUDPAddr      func(network, address string) (*net.UDPAddr, error)
	handleUDPAddrFromAddrPort func(addr netip.AddrPort) *net.UDPAddr
}

func NewUDPMockNetAddr() *mockUDPNetAddrImpl {
	mNetAddrImpl := mockUDPNetAddrImpl{}
	return &mNetAddrImpl
}

func (m *mockUDPNetAddrImpl) String() string {
	return m.handleString()
}

func (m *mockUDPNetAddrImpl) Network() string {
	return m.handleNetwork()
}

func (m *mockUDPNetAddrImpl) AddrPort() netip.AddrPort {
	return m.handleAddrPort()
}
func (m *mockUDPNetAddrImpl) ResolveUDPAddr(network, address string) (*net.UDPAddr, error) {
	return m.handleResolveUDPAddr(network, address)
}
func (m *mockUDPNetAddrImpl) UDPAddrFromAddrPort(addr netip.AddrPort) *net.UDPAddr {
	return m.handleUDPAddrFromAddrPort(addr)
}
