package hypertrace

import (
	"fmt"
	"github.com/sirupsen/logrus"
)

const (
	UploadTokenLength = 16
)

var (
	ErrRegisterUIDError = fmt.Errorf("uid registration error")
	ErrUIDNotFound      = fmt.Errorf("uid not found")
	ErrTokenNotFound    = fmt.Errorf("token not found")
)

type ITracing interface {
	RegisterNewTraceUser(UID string) (err error)
	GetUploadToken(UID, secret string) (token string, err error)
	SaveTraceData(UID, uploadToken string, data []*TraceData) (err error)
	PurgeOldTraceData(oldestTimeStamp int64, uploadToken string) (err error)
	GetTraceData(UID, uploadToken string) (traces []*TraceData, err error)
}

type TraceUser struct {
	UID    string
	Traces []*TraceData
}

type TraceData struct {
	ContactUID string `json:"contactUid"`
	Timestamp  int64  `json:"timestamp"`
	ModelC     string `json:"modelC"`
	ModelP     string `json:"modelP"`
	RSSI       int    `json:"rssi"`
	TxPower    int    `json:"txPower"`
	Org        string `json:"org"`
}

type DataUpload struct {
	UID         string               `json:"uid"`
	UploadToken string               `json:"uploadToken"`
	Traces      []*UploadTraceRecord `json:"traces"`
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

func NewInMemoryTracing() ITracing {
	tracing := &InMemoryTracing{
		UIDs:           make(map[string]*TraceUser),
		SecretTokenMap: make(map[string]string),
	}

	tracing.SecretTokenMap["secret1"] = "UAMnfvgwsXW96kZu"
	tracing.SecretTokenMap["secret2"] = "L6Z1dudr9908ywhk"
	tracing.SecretTokenMap["secret3"] = "RKX8aJJGJE113Uto"
	tracing.SecretTokenMap["secret4"] = "TDy0gnzJbDx8mNSN"
	tracing.SecretTokenMap["secret5"] = "qxCSSnT1qt1XPU6D"
	return tracing
}

type InMemoryTracing struct {
	UIDs           map[string]*TraceUser
	SecretTokenMap map[string]string
}

func (trace *InMemoryTracing) RegisterNewTraceUser(UID string) (err error) {
	if _, ok := trace.UIDs[UID]; !ok {
		trace.UIDs[UID] = &TraceUser{
			UID:    UID,
			Traces: make([]*TraceData, 0),
		}
	}
	return nil
}
func (trace *InMemoryTracing) GetUploadToken(uid, secret string) (token string, err error) {
	logrus.Infof("UID = %s", uid)
	if _, ok := trace.UIDs[uid]; !ok {
		return "", fmt.Errorf("%w : %s", ErrUIDNotFound, uid)
	}
	logrus.Infof("Secret = %s", secret)

	if td, ok := trace.SecretTokenMap[secret]; ok {
		return td, nil
	}
	return "", ErrTokenNotFound
}
func (trace *InMemoryTracing) SaveTraceData(UID, uploadToken string, data []*TraceData) (err error) {
	found := false
	for _, v := range trace.SecretTokenMap {
		if v == uploadToken {
			found = true
		}
	}
	if !found {
		return ErrTokenNotFound
	}
	if td, ok := trace.UIDs[UID]; ok {
		td.Traces = append(td.Traces, data...)
		return nil
	}
	return fmt.Errorf("%w : %s", ErrUIDNotFound, UID)
}
func (trace *InMemoryTracing) PurgeOldTraceData(oldestTimeStamp int64, uploadToken string) (err error) {
	found := false
	for _, v := range trace.SecretTokenMap {
		if v == uploadToken {
			found = true
		}
	}
	if !found {
		return ErrTokenNotFound
	}
	for _, tUser := range trace.UIDs {
		newTraceData := make([]*TraceData, 0)
		for _, td := range tUser.Traces {
			if td.Timestamp >= oldestTimeStamp {
				newTraceData = append(newTraceData, td)
			}
		}
		tUser.Traces = newTraceData
	}
	return nil
}
func (trace *InMemoryTracing) GetTraceData(UID, uploadToken string) (traces []*TraceData, err error) {
	found := false
	for _, v := range trace.SecretTokenMap {
		if v == uploadToken {
			found = true
		}
	}
	if !found {
		return nil, ErrTokenNotFound
	}
	if td, ok := trace.UIDs[UID]; ok {
		return td.Traces, nil
	}
	return nil, fmt.Errorf("%w : %s", ErrUIDNotFound, UID)
}
