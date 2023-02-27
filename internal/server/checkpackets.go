package server

import (
	"encoding/binary"
)

type mPacket int

func getType(m []byte) uint16 {
	return binary.BigEndian.Uint16(m)
}

func getLength(m []byte) uint16 {
	return binary.BigEndian.Uint16(m)
}

func getMagicCookie(m []byte) uint32 {
	return binary.BigEndian.Uint32(m)
}

func getTransactionID(m []byte) []byte {
	return m
}

const headerlen = 20
const errorcodelen = 8

func (mPkt *mPacket) CheckHeader(m []byte) (uint16, uint16, uint32, []byte) {
	mBufPtr := *mPkt
	mType := getType(m[mBufPtr:2])
	mLength := getLength(m[mBufPtr+2 : 4])
	mCookie := getMagicCookie(m[mBufPtr+4 : 8])
	mTransactionID := getTransactionID(m[mBufPtr+8 : 20])
	*mPkt += 8 // don't increment 12 as for some reason the code is adding the transaction ID twice to the buffer..
	return mType, mLength, mCookie, mTransactionID
}

func (mPkt *mPacket) CheckBadAuthHeader(m []byte) (bRet bool) {
	bRet = false
	mType, mLength, mCookie, mTransactionID := mPkt.CheckHeader(m)
	if mType == 0x113 && mLength == 0x3c && mCookie != 0x0 && len(mTransactionID) == 12 {
		bRet = true
	}
	return bRet
}

func (mPkt *mPacket) CheckTransactionID(m []byte) (bRet bool) {
	mBufPtr := *mPkt
	mTransactionID := getTransactionID(m[mBufPtr : mBufPtr+12])
	if string(mTransactionID) == "00000000000000" {
		bRet = true
	}
	*mPkt += 12
	return bRet
}

func (mPkt *mPacket) CheckMessageType(m []byte, mRequiredType uint16) (bRet bool) {
	mBufPtr := *mPkt
	mMessageType := binary.BigEndian.Uint16(m[mBufPtr : mBufPtr+2])
	if mMessageType == mRequiredType {
		bRet = true
	}
	*mPkt += 2
	return bRet
}

func (mPkt *mPacket) CheckErrorCode(m []byte, mRequiredType uint8, mRequiredLength uint8, mRequiredCode uint32) (bRet bool) {
	// find type
	mtype := binary.BigEndian.Uint16(m[*mPkt : *mPkt+2])
	if mtype != uint16(mRequiredType) {
		return false
	}
	// find length
	mLength := binary.BigEndian.Uint16(m[*mPkt+2 : *mPkt+4])
	if mLength != uint16(mRequiredLength) {
		return false
	}
	mErrorCode := binary.BigEndian.Uint32(m[*mPkt+4 : *mPkt+8])
	if mErrorCode == mRequiredCode {
		bRet = true
	}
	*mPkt += 8
	return bRet
}

func (mPkt *mPacket) CheckNonce(m []byte) (bRet bool, nonce []byte) {
	// find type
	mtype := binary.BigEndian.Uint16(m[*mPkt : *mPkt+2])
	if mtype != 0x15 {
		return false, nil
	}
	// find length
	mLength := binary.BigEndian.Uint16(m[*mPkt+2 : *mPkt+4])
	if mLength != 0x20 {
		return false, nil
	}
	/*nonce = make([]byte, 0x20)
	copy(nonce, m[*mPkt+4:*mPkt + 0x24])*/
	nonce = m[*mPkt+4 : *mPkt+0x24]
	*mPkt += mPacket(mLength) + 4
	return true, nonce
}

func (mPkt *mPacket) CheckRealm(m []byte, mval string) (bRet bool) {
	mBufPtr := *mPkt

	// find type
	mtype := binary.BigEndian.Uint16(m[*mPkt : *mPkt+2])
	if mtype != 0x14 {
		return false
	}
	// find length
	mLength := binary.BigEndian.Uint16(m[mBufPtr+2 : mBufPtr+4])
	if mLength != uint16(len(mval)) {
		return false
	}

	mEndPtr := mBufPtr + 4 + mPacket(len(mval))
	if mval != string(m[mBufPtr+4:mEndPtr]) {
		return false
	}
	*mPkt += mPacket(mLength) + 4
	return true
}

func (mPkt *mPacket) GetNonceFromResponse(mBadAuthPacket []byte) []byte {
	mBufPtr := *mPkt + (headerlen + errorcodelen)
	bOK, nonce := mBufPtr.CheckNonce(mBadAuthPacket)
	if !bOK {
		return nil
	}
	return nonce
}
