//go:build !js
// +build !js

package server

import (
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/pion/logging"
	"github.com/pion/stun"
	"github.com/pion/turn/v2/internal/allocation"
	"github.com/pion/turn/v2/internal/proto"
	"github.com/stretchr/testify/assert"
)

func TestAllocationLifeTime(t *testing.T) {
	t.Run("Parsing", func(t *testing.T) {
		lifetime := proto.Lifetime{
			Duration: 5 * time.Second,
		}

		m := &stun.Message{}
		lifetimeDuration := allocationLifeTime(m)

		if lifetimeDuration != proto.DefaultLifetime {
			t.Errorf("Allocation lifetime should be default time duration")
		}

		assert.NoError(t, lifetime.AddTo(m))

		lifetimeDuration = allocationLifeTime(m)
		if lifetimeDuration != lifetime.Duration {
			t.Errorf("Expect lifetimeDuration is %s, but %s", lifetime.Duration, lifetimeDuration)
		}
	})

	// If lifetime is bigger than maximumLifetime
	t.Run("Overflow", func(t *testing.T) {
		lifetime := proto.Lifetime{
			Duration: maximumAllocationLifetime * 2,
		}

		m2 := &stun.Message{}
		_ = lifetime.AddTo(m2)

		lifetimeDuration := allocationLifeTime(m2)
		if lifetimeDuration != proto.DefaultLifetime {
			t.Errorf("Expect lifetimeDuration is %s, but %s", proto.DefaultLifetime, lifetimeDuration)
		}
	})

	t.Run("DeletionZeroLifetime", func(t *testing.T) {
		l, err := net.ListenPacket("udp4", "0.0.0.0:0")
		assert.NoError(t, err)
		defer func() {
			assert.NoError(t, l.Close())
		}()

		logger := logging.NewDefaultLoggerFactory().NewLogger("turn")

		allocationManager, err := allocation.NewManager(allocation.ManagerConfig{
			AllocatePacketConn: func(network string, requestedPort int) (net.PacketConn, net.Addr, error) {
				conn, listenErr := net.ListenPacket(network, "0.0.0.0:0")
				if err != nil {
					return nil, nil, listenErr
				}

				return conn, conn.LocalAddr(), nil
			},
			AllocateConn: func(network string, requestedPort int) (net.Conn, net.Addr, error) {
				return nil, nil, nil
			},
			LeveledLogger: logger,
		})
		assert.NoError(t, err)

		staticKey := []byte("ABC")
		r := Request{
			AllocationManager: allocationManager,
			Nonces:            &sync.Map{},
			Conn:              l,
			SrcAddr:           &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5000},
			Log:               logger,
			AuthHandler: func(username string, realm string, srcAddr net.Addr) (key []byte, ok bool) {
				return staticKey, true
			},
		}
		r.Nonces.Store(string(staticKey), time.Now())

		fiveTuple := &allocation.FiveTuple{SrcAddr: r.SrcAddr, DstAddr: r.Conn.LocalAddr(), Protocol: allocation.UDP}

		_, err = r.AllocationManager.CreateAllocation(fiveTuple, r.Conn, 0, time.Hour)
		assert.NoError(t, err)

		assert.NotNil(t, r.AllocationManager.GetAllocation(fiveTuple))

		m := &stun.Message{}
		assert.NoError(t, (proto.Lifetime{}).AddTo(m))
		assert.NoError(t, (stun.MessageIntegrity(staticKey)).AddTo(m))
		assert.NoError(t, (stun.Nonce(staticKey)).AddTo(m))
		assert.NoError(t, (stun.Realm(staticKey)).AddTo(m))
		assert.NoError(t, (stun.Username(staticKey)).AddTo(m))

		//assert.NoError(t, handleRefreshRequest(r, m))
		assert.Nil(t, r.AllocationManager.GetAllocation(fiveTuple))
	})
}

func setupRequestPacket(t *testing.T, realm string, mockStunMessageImpl *mockStunMessageImpl) (Request, *mockPacketConnImpl, error) {
	var mBuff = []byte("It's another packet")
	mSrcNetAddr := net.UDPAddr{
		IP:   net.IPv4(byte(0xfa), byte(0x80), byte(0x0), byte(0x0)),
		Port: 5600,
	}
	mMockNetAddrImpl := makeMockNetAddr(t, "127.0.0.1:80")
	mLogging := NewLeveledLogger()

	mMockConnPacketImpl := NewMockPacketConnImpl()
	mMockConnPacketImpl.handleLocalAddr = func() net.Addr {
		mNewAddr := mMockNetAddrImpl
		return mNewAddr
	}

	mNonces := sync.Map{}
	mNonces.Store("456", "Hello")
	allocationManager, err := allocation.NewManager(allocation.ManagerConfig{
		AllocatePacketConn: func(network string, requestedPort int) (net.PacketConn, net.Addr, error) {
			conn, listenErr := net.ListenPacket(network, "0.0.0.0:0")
			if listenErr != nil {
				return nil, nil, listenErr
			}

			return conn, conn.LocalAddr(), nil
		},
		AllocateConn: func(network string, requestedPort int) (net.Conn, net.Addr, error) {
			return nil, nil, nil
		},
		LeveledLogger: mLogging,
	})
	if err != nil {
		return Request{}, nil, err
	}

	r := Request{
		AllocationManager: allocationManager,
		Conn:              mMockConnPacketImpl,
		Buff:              mBuff,
		SrcAddr:           &mSrcNetAddr,
		Log:               mLogging,
		Nonces:            &mNonces,
		Realm:             realm,
		AuthHandler:       mockStunMessageImpl.handleAuth,
	}
	fiveTuple := &allocation.FiveTuple{SrcAddr: r.SrcAddr, DstAddr: r.Conn.LocalAddr(), Protocol: allocation.UDP}

	_, err = r.AllocationManager.CreateAllocation(fiveTuple, r.Conn, 0, time.Hour)
	assert.NoError(t, err)
	return r, mMockConnPacketImpl, nil
}

func Test_handleAllocateRequestBlankMsgNoAuthCredentials(t *testing.T) {
	var mBadAuthPacket []byte
	mockStunImpl := NewMockStunImpl()
	mockStunMessageImpl := &mockStunImpl.Message
	var m mockStunMessageIF = mockStunMessageImpl
	mockStunMessageImpl.handleGetMessage = func() *stun.Message {
		return &stun.Message{}
	}
	mockStunMessageImpl.handleContains = func(t stun.AttrType) bool {
		return false
	}
	mockStunMessageImpl.handleGetTransactionID = func() [stun.TransactionIDSize]byte {
		var mTransactionID [stun.TransactionIDSize]byte = [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
		// this is pretend.. we're not actually checking the data in the message
		return mTransactionID
	}
	r, mMockConnPacketImpl, err := setupRequestPacket(t, "MyEnterprise", mockStunMessageImpl)
	if err != nil {
		fmt.Printf("Error %s\n", err.Error())
		t.FailNow()
	}
	mMockConnPacketImpl.handleWriteTo = func(p []byte, addr net.Addr) (n int, err error) {
		mBadAuthPacket = p
		return 0, nil
	}

	err = handleAllocateRequest(r, m)
	fmt.Printf("Sent %v\n", mBadAuthPacket)
	var mSentPacket mPacket = 0
	var mTypePacket mPacket = 0
	// note the SendPacket is a incrementing pointer into the message packet.
	bHeaderOK := mSentPacket.CheckBadAuthHeader(mBadAuthPacket)
	bTXID := mSentPacket.CheckTransactionID(mBadAuthPacket)
	bMsgTypeOK := mTypePacket.CheckMessageType(mBadAuthPacket, 0x113)
	mRequiredCode := uint32(0x401)
	mRequiredLength := uint8(0x04)
	mRequiredType := uint8(0x09)
	bErrorCodeOK := mSentPacket.CheckErrorCode(mBadAuthPacket, mRequiredType, mRequiredLength, mRequiredCode)
	bNonceOK, _ := mSentPacket.CheckNonce(mBadAuthPacket) //can't check actual nonce - as it's time dependent... can only check for non zero presence..
	bRealmOK := mSentPacket.CheckRealm(mBadAuthPacket, "MyEnterprise")
	if !bHeaderOK || !bTXID || !bMsgTypeOK || !bErrorCodeOK || !bNonceOK || !bRealmOK {
		fmt.Printf("All Fine\n")
	}
	if err != nil {
		fmt.Printf("Error %s\n", err.Error())
	}
}

func Test_handleAllocateRequestBlankMsgSupplyAuthCredentials(t *testing.T) {
	var mBadAuthPacket []byte
	var username stun.Username          // read-only
	var password string                 // read-only
	var realm stun.Realm                // read-only
	var integrity stun.MessageIntegrity // read-only
	mockStunImpl := NewMockStunImpl()
	mockStunMessageImpl := &mockStunImpl.Message
	// must instantiate this function before we setup the request packet in order that AuthHandler has a non nil address
	mockStunMessageImpl.handleAuth = func(username string, realm string, srcAddr net.Addr) (key []byte, ok bool) {
		// MD5 hash on user:realm:pass - reference page 35 RFC5389...
		ourkey := stun.NewLongTermIntegrity(
			string(username), string(realm), string(password),
		)
		return ourkey, true
	}

	var m mockStunMessageIF = mockStunMessageImpl
	username = []byte("dummyuser")
	password = "password"
	realm = []byte("myenterprise")
	integrity = stun.NewLongTermIntegrity(
		string(username), string(realm), string(password),
	)

	r, mMockConnPacketImpl, err := setupRequestPacket(t, string(realm), mockStunMessageImpl)
	if err != nil {
		fmt.Printf("Error %s\n", err.Error())
		t.FailNow()
	}
	mockStunMessageImpl.handleGetMessage = func() *stun.Message {
		// needs to have nonce, username & integrity in it - in order to pass the checks in authenticateRequest
		msg := &stun.Message{}
		nonce := stun.NewNonce("123")
		requested_transport := proto.RequestedTransport{
			Protocol: proto.ProtoUDP,
		}
		// note this gives us the chance within a test routine to deliberately set or omit particular client message attributes
		msg.Build(
			integrity,
			nonce,
			username,
			realm,
			requested_transport,
		)
		return msg
	}

	mockStunMessageImpl.handleBuild = func(setters ...stun.Setter) error {
		return nil
	}

	mockStunMessageImpl.handleContains = func(t stun.AttrType) bool {
		// this is pretend.. we're not actually checking the data in the message
		var bContains bool
		switch t {
		case stun.AttrDontFragment:
			bContains = false
		default:
			bContains = true
		}
		return bContains
	}
	mockStunMessageImpl.handleGetTransactionID = func() [stun.TransactionIDSize]byte {
		var mTransactionID [stun.TransactionIDSize]byte = [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
		// this is pretend.. we're not actually checking the data in the message
		return mTransactionID
	}

	mMockConnPacketImpl.handleWriteTo = func(p []byte, addr net.Addr) (n int, err error) {
		mBadAuthPacket = p
		return 0, nil
	}
	var blankMsg []byte
	m.SetMessage(blankMsg)
	// include a message integrity attr...
	r.Nonces.Store("123", time.Now())

	err = handleAllocateRequest(r, m)
	fmt.Printf("\nSent %v\n", mBadAuthPacket)
	if err != nil {
		fmt.Printf("Error %s\n", err.Error())
		t.FailNow()
	}

}

func Test_handleRefreshRequestPeerMismatch(t *testing.T) {
	var mBadAuthPacket []byte
	var username stun.Username          // read-only
	var password string                 // read-only
	var realm stun.Realm                // read-only
	var integrity stun.MessageIntegrity // read-only
	mockStunImpl := NewMockStunImpl()
	mockStunMessageImpl := &mockStunImpl.Message
	// must instantiate this function before we setup the request packet in order that AuthHandler has a non nil address
	mockStunMessageImpl.handleAuth = func(username string, realm string, srcAddr net.Addr) (key []byte, ok bool) {
		// MD5 hash on user:realm:pass - reference page 35 RFC5389...
		ourkey := stun.NewLongTermIntegrity(
			string(username), string(realm), string(password),
		)
		return ourkey, true
	}

	var m mockStunMessageIF = mockStunMessageImpl
	username = []byte("dummyuser")
	password = "password"
	realm = []byte("myenterprise")
	integrity = stun.NewLongTermIntegrity(
		string(username), string(realm), string(password),
	)

	r, mMockConnPacketImpl, err := setupRequestPacket(t, string(realm), mockStunMessageImpl)
	if err != nil {
		fmt.Printf("Error %s\n", err.Error())
		t.FailNow()
	}
	mockStunMessageImpl.handleGetMessage = func() *stun.Message {
		// needs to have nonce, username & integrity in it - in order to pass the checks in authenticateRequest
		msg := &stun.Message{}
		nonce := stun.NewNonce("123")
		requested_transport := proto.RequestedTransport{
			Protocol: proto.ProtoUDP,
		}
		// note this gives us the chance within a test routine to deliberately set or omit particular client message attributes
		msg.Build(
			integrity,
			nonce,
			username,
			realm,
			requested_transport,
		)
		return msg
	}

	mockStunMessageImpl.handleBuild = func(setters ...stun.Setter) error {
		return nil
	}

	mockStunMessageImpl.handleContains = func(t stun.AttrType) bool {
		// this is pretend.. we're not actually checking the data in the message
		var bContains bool
		switch t {
		case stun.AttrDontFragment:
			bContains = false
		default:
			bContains = true
		}
		return bContains
	}
	mockStunMessageImpl.handleGetTransactionID = func() [stun.TransactionIDSize]byte {
		var mTransactionID [stun.TransactionIDSize]byte = [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
		// this is pretend.. we're not actually checking the data in the message
		return mTransactionID
	}

	mMockConnPacketImpl.handleWriteTo = func(p []byte, addr net.Addr) (n int, err error) {
		mBadAuthPacket = p
		return 0, nil
	}
	var blankMsg []byte
	m.SetMessage(blankMsg)
	// include a message integrity attr...
	r.Nonces.Store("123", time.Now())

	err = handleRefreshRequest(r, m)
	fmt.Printf("\nSent %v\n", mBadAuthPacket)
	if err != nil {
		fmt.Printf("Error %s\n", err.Error())
		t.FailNow()
	}

}
