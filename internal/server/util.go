package server

import (
	"crypto/md5" //nolint:gosec,gci
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/pion/stun"
	"github.com/pion/turn/v2/internal/proto"
)

const (
	maximumAllocationLifetime = time.Hour // https://tools.ietf.org/html/rfc5766#section-6.2 defines 3600 seconds recommendation
	nonceLifetime             = time.Hour // https://tools.ietf.org/html/rfc5766#section-4

)

func randSeq(n int) string {
	letters := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] //nolint:gosec
	}
	return string(b)
}

func buildNonce() (string, error) {
	/* #nosec */
	h := md5.New()
	if _, err := io.WriteString(h, strconv.FormatInt(time.Now().Unix(), 10)); err != nil {
		return "", fmt.Errorf("%w: %v", errFailedToGenerateNonce, err)
	}
	if _, err := io.WriteString(h, strconv.FormatInt(rand.Int63(), 10)); err != nil { //nolint:gosec
		return "", fmt.Errorf("%w: %v", errFailedToGenerateNonce, err)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func buildAndSend(conn net.PacketConn, dst net.Addr, attrs ...stun.Setter) error {
	msg, err := stun.Build(attrs...)
	if err != nil {
		return err
	}
	_, err = conn.WriteTo(msg.Raw, dst)
	return err
}

// Send a STUN packet and return the original error to the caller
func buildAndSendErr(conn net.PacketConn, dst net.Addr, err error, attrs ...stun.Setter) error {
	if sendErr := buildAndSend(conn, dst, attrs...); sendErr != nil {
		err = fmt.Errorf("%w %v %v", errFailedToSendError, sendErr, err)
	}
	return err
}

func buildMsg(transactionID [stun.TransactionIDSize]byte, msgType stun.MessageType, additional ...stun.Setter) []stun.Setter {
	return append([]stun.Setter{&stun.Message{TransactionID: transactionID}, msgType}, additional...)
}

// we're forced to change the method signature here to accept a StunMessage interface rather than the struct... and follow through
// the consequences of this..
// the consequences at phase 1 have been minimised by invoking m.GetMessage(). This enables the signature of called functions/methods to remain
// unchanged. In the longer run these would also change to accept an interface, rather than a struct.
func authenticateRequest(r Request, m stun.StunMessageIF, callingMethod stun.Method) (stun.MessageIntegrity, bool, error) {
	//func authenticateRequest(r Request, m *stun.Message, callingMethod stun.Method) (stun.MessageIntegrity, bool, error) {
	respondWithNonce := func(responseCode stun.ErrorCode) (stun.MessageIntegrity, bool, error) {
		nonce, err := buildNonce()
		if err != nil {
			return nil, false, err
		}

		// Nonce has already been taken
		if _, keyCollision := r.Nonces.LoadOrStore(nonce, time.Now()); keyCollision {
			return nil, false, errDuplicatedNonce
		}

		return nil, false, buildAndSend(r.Conn, r.SrcAddr, buildMsg(m.GetTransactionID(),
			stun.NewType(callingMethod, stun.ClassErrorResponse),
			&stun.ErrorCodeAttribute{Code: responseCode},
			stun.NewNonce(nonce),
			stun.NewRealm(r.Realm),
		)...)
	}

	if !m.Contains(stun.AttrMessageIntegrity) {
		return respondWithNonce(stun.CodeUnauthorized)
	}

	nonceAttr := &stun.Nonce{}
	usernameAttr := &stun.Username{}
	realmAttr := &stun.Realm{}
	badRequestMsg := buildMsg(m.GetTransactionID(), stun.NewType(callingMethod, stun.ClassErrorResponse), &stun.ErrorCodeAttribute{Code: stun.CodeBadRequest})

	if err := nonceAttr.GetFrom(m.GetMessage()); err != nil {
		return nil, false, buildAndSendErr(r.Conn, r.SrcAddr, err, badRequestMsg...)
	}

	// Assert Nonce exists and is not expired
	nonceCreationTime, nonceFound := r.Nonces.Load(string(*nonceAttr))
	if !nonceFound {
		r.Nonces.Delete(nonceAttr)
		return respondWithNonce(stun.CodeStaleNonce)
	}

	if timeValue, ok := nonceCreationTime.(time.Time); !ok || time.Since(timeValue) >= nonceLifetime {
		r.Nonces.Delete(nonceAttr)
		return respondWithNonce(stun.CodeStaleNonce)
	}

	if err := realmAttr.GetFrom(m.GetMessage()); err != nil {
		return nil, false, buildAndSendErr(r.Conn, r.SrcAddr, err, badRequestMsg...)
	} else if err := usernameAttr.GetFrom(m.GetMessage()); err != nil {
		return nil, false, buildAndSendErr(r.Conn, r.SrcAddr, err, badRequestMsg...)
	}

	ourKey, ok := r.AuthHandler(usernameAttr.String(), realmAttr.String(), r.SrcAddr)
	if !ok {
		return nil, false, buildAndSendErr(r.Conn, r.SrcAddr, fmt.Errorf("%w %s", errNoSuchUser, usernameAttr.String()), badRequestMsg...)
	}

	if err := stun.MessageIntegrity(ourKey).Check(m.GetMessage()); err != nil {
		return nil, false, buildAndSendErr(r.Conn, r.SrcAddr, err, badRequestMsg...)
	}

	return stun.MessageIntegrity(ourKey), true, nil
}

func allocationLifeTime(m *stun.Message) time.Duration {
	lifetimeDuration := proto.DefaultLifetime

	var lifetime proto.Lifetime
	if err := lifetime.GetFrom(m); err == nil {
		if lifetime.Duration < maximumAllocationLifetime {
			lifetimeDuration = lifetime.Duration
		}
	}

	return lifetimeDuration
}
