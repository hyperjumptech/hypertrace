package hypertrace

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	UploadTokenLength = 16
)

var (
	ErrRegisterUIDError = fmt.Errorf("uid registration error")
	ErrUIDNotFound      = fmt.Errorf("uid not found")
	ErrTokenNotFound    = fmt.Errorf("token not found")
	ErrSecretNotValid   = fmt.Errorf("secret not valid")
	ErrInvalidParameter = fmt.Errorf("invalid parameter")
)

type ITracing interface {
	RegisterNewUser(ctx context.Context, UID, PIN string) (err error)
	GetHandshakePIN(ctx context.Context, UID string) (PIN string, err error)

	SaveTraceData(ctx context.Context, UID, OID string, data []*TraceData) (err error)
	PurgeOldTraceData(ctx context.Context, oldestTimeStamp int64) (err error)
	GetTraceData(ctx context.Context, UID string) (traces []*TraceData, err error)

	RegisterNewOfficer(ctx context.Context, OID, secret string) (err error)
	GetOfficerID(ctx context.Context, secret string) (OID string, err error)
	DeleteOfficer(ctx context.Context, OID string) (err error)
}

type User struct {
	UID string `json:"uid" bson:"uid"`
	PIN string `json:"pin" bson:"pin"`
}

type Officer struct {
	OID    string `json:"oid" bson:"oid"`
	Secret string `json:"secret" bson:"secret"`
}

type TraceData struct {
	OID       string `json:"oid,omitempty" bson:"oid"`
	UID       string `json:"uid,omitempty" bson:"uid"`
	CUID      string `json:"cuid" bson:"cuid"`
	Timestamp int64  `json:"timestamp" bson:"timestamp"`
	ModelC    string `json:"modelC" bson:"modelC"`
	ModelP    string `json:"modelP" bson:"modelP"`
	RSSI      int    `json:"rssi" bson:"rssi"`
	TxPower   int    `json:"txPower" bson:"txPower"`
	Org       string `json:"org" bson:"org"`
}

func NewUploadToken(uid, oid string, validHour int) *UploadToken {
	return &UploadToken{
		OID:        oid,
		UID:        uid,
		ValidFrom:  time.Now().Unix(),
		ValidUntil: time.Now().Add(time.Duration(validHour) * time.Hour).Unix(),
	}
}

func NewUploadTokenFromString(token string, key []byte) (*UploadToken, error) {
	dataJson, err := decodeAndDecrypt(token, key)
	if err != nil {
		return nil, err
	}
	ut := &UploadToken{}
	err = json.Unmarshal(dataJson, ut)
	return ut, err
}

type UploadToken struct {
	OID        string `json:"oid" bson:"oid"`
	UID        string `json:"uid" bson:"uid"`
	ValidFrom  int64  `json:"nbf" bson:"nbf"`
	ValidUntil int64  `json:"exp" bson:"exp"`
}

func (ut *UploadToken) IsValid() bool {
	n := time.Now().Unix()
	return n > ut.ValidFrom && n < ut.ValidUntil
}

func (ut *UploadToken) ToToken(key []byte) (token string, err error) {
	utBytes, err := json.Marshal(ut)
	if err != nil {
		return "", err
	}
	return encryptAndEncode(utBytes, key)
}

type DataUpload struct {
	UID         string               `json:"uid" bson:"uid"`
	UploadToken string               `json:"uploadToken" bson:"uploadToken"`
	Traces      []*UploadTraceRecord `json:"traces" bson:"traces"`
}

type UploadTraceRecord struct {
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"msg"`
	ModelC    string `json:"modelC"`
	ModelP    string `json:"modelP"`
	RSSI      int    `json:"rssi"`
	TxPower   int    `json:"txPower"`
	Org       string `json:"org"`
}
