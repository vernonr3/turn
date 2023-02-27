// Package server implements the private API to implement a TURN server
package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pion/logging"
	"github.com/pion/stun"
	"github.com/pion/turn/v2/internal/allocation"
	"github.com/pion/turn/v2/internal/proto"
)

// Request contains all the state needed to process a single incoming datagram
type Request struct {
	// Current Request State
	Conn    net.PacketConn
	SrcAddr net.Addr
	Buff    []byte

	// Server State
	AllocationManager *allocation.Manager
	Nonces            *sync.Map

	// User Configuration
	AuthHandler        func(username string, realm string, srcAddr net.Addr) (key []byte, ok bool)
	Log                logging.LeveledLogger
	Realm              string
	ChannelBindTimeout time.Duration
}

var ctx context.Context

// HandleRequest processes the give Request
func HandleRequest(r Request) error {
	//var task *trace.Task

	//ctx, task = trace.NewTask(nil, "Request")
	r.Log.Debugf("received %d bytes of udp from %s on %s", len(r.Buff), r.SrcAddr.String(), r.Conn.LocalAddr().String())

	if proto.IsChannelData(r.Buff) {
		return handleDataPacket(r)
	}
	err := handleTURNPacket(r)
	//task.End()
	return err
}

func handleDataPacket(r Request) error {
	r.Log.Debugf("received DataPacket from %s", r.SrcAddr.String())
	c := proto.ChannelData{Raw: r.Buff}
	if err := c.Decode(); err != nil {
		return fmt.Errorf("%w: %v", errFailedToCreateChannelData, err)
	}

	err := handleChannelData(r, &c)
	if err != nil {
		err = fmt.Errorf("%w from %v: %v", errUnableToHandleChannelData, r.SrcAddr, err)
	}

	return err
}

func handleTURNPacket(r Request) error {
	var m = stun.NewStunMessage()
	r.Log.Debug("handleTURNPacket")
	m.ApplyBuf(r.Buff...)
	r.Log.Debug(fmt.Sprintf("%v\n", r.Buff))
	///m := &stun.Message{Raw: append([]byte{}, r.Buff...)}
	if err := m.Decode(); err != nil {
		return fmt.Errorf("%w: %v", errFailedToCreateSTUNPacket, err)
	}
	mTypeMethod := m.GetTypeMethod()
	mTypeClass := m.GetTypeClass()
	h, err := getMessageHandler(mTypeClass, mTypeMethod, r.Log)
	if err != nil {
		return fmt.Errorf("%w %v-%v from %v: %v", errUnhandledSTUNPacket, mTypeMethod, mTypeClass, r.SrcAddr, err)
	}
	//endRegion := trace.StartRegion(ctx, "Handle Message")
	err = h(r, m)
	if err != nil {
		return fmt.Errorf("%w %v-%v from %v: %v", errFailedToHandle, mTypeMethod, mTypeClass, r.SrcAddr, err)
	}
	//endRegion.End()
	return nil
}

// func getMessageHandler(class stun.MessageClass, method stun.Method) (func(r Request, m *stun.Message) error, error) {
// we're forced to change the function signature in order that we can invoke the methods in turn.go from the turn_test.go
// with an interface to stunMessage	rather than the struct itself.
func getMessageHandler(class stun.MessageClass, method stun.Method, logger logging.LeveledLogger) (func(r Request, m stun.StunMessageIF) error, error) {
	switch class {
	case stun.ClassIndication:
		switch method {
		case stun.MethodSend:
			return handleSendIndication, nil
		default:
			return nil, fmt.Errorf("%w: %s", errUnexpectedMethod, method)
		}

	case stun.ClassRequest:
		switch method {
		case stun.MethodAllocate:
			logger.Debug("Method Allocate")
			return handleAllocateRequest, nil
		case stun.MethodRefresh:
			logger.Debug("Method Refresh")
			return handleRefreshRequest, nil
		case stun.MethodCreatePermission:
			logger.Debug("Method Create Permission")
			return handleCreatePermissionRequest, nil
		case stun.MethodChannelBind:
			logger.Debug("Method Channel Bind")
			return handleChannelBindRequest, nil
		case stun.MethodBinding:
			logger.Debug("Method Binding")
			return handleBindingRequest, nil
		default:
			return nil, fmt.Errorf("%w: %s", errUnexpectedMethod, method)
		}

	default:
		return nil, fmt.Errorf("%w: %s", errUnexpectedClass, class)
	}
}
