package server

import (
	"github.com/pion/stun"
	"github.com/pion/turn/v2/internal/ipnet"
)

// func handleBindingRequest(r Request, m *stun.Message) error {
func handleBindingRequest(r Request, m stun.StunMessageIF) error {
	r.Log.Debugf("received BindingRequest from %s", r.SrcAddr.String())

	ip, port, err := ipnet.AddrIPPort(r.SrcAddr)
	if err != nil {
		return err
	}

	attrs := buildMsg(m.GetTransactionID(), stun.BindingSuccess, &stun.XORMappedAddress{
		IP:   ip,
		Port: port,
	}, stun.Fingerprint)

	return buildAndSend(r.Conn, r.SrcAddr, attrs...)
}
