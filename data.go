package hypertrace

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	UploadTokenLength = 16
)

var (
	ErrRegisterUIDError = fmt.Errorf("uid registration error")
	ErrUIDNotFound      = fmt.Errorf("uid not found")
	ErrTokenNotFound    = fmt.Errorf("token not found")
	inMemoryLog         = logrus.WithField("DB", "InMemory")
	mongoLog            = logrus.WithField("DB", "MongoDB")
)

type ITracing interface {
	RegisterNewUser(ctx context.Context, UID, PIN string) (err error)
	GetHandshakePIN(ctx context.Context, UID string) (PIN string, err error)
	UserPIN(ctx context.Context, UID, PIN, NewPIN string) (err error)

	GetUploadToken(ctx context.Context, UID, secret string) (token string, err error)
	SaveTraceData(ctx context.Context, UID, uploadToken string, data []*TraceData) (err error)
	PurgeOldTraceData(ctx context.Context, oldestTimeStamp int64, secret string) (err error)
	GetTraceData(ctx context.Context, UID, secret string) (traces []*TraceData, err error)

	RegisterNewOfficer(ctx context.Context, officerID, secret string) (err error)
	GetOfficerID(ctx context.Context, secret string) (officerID string, err error)
	DeleteOfficer(ctx context.Context, officerID string) (err error)
}

type User struct {
	UID string `json:"uid"`
	PIN string `json:"pin"`
}

type Officer struct {
	OID    string `json:"oid"`
	Secret string `json:"secret"`
}

type TraceData struct {
	OID       string `json:"oid,omitempty"`
	UID       string `json:"uid,omitempty"`
	CUID      string `json:"cuid"`
	Timestamp int64  `json:"timestamp"`
	ModelC    string `json:"modelC"`
	ModelP    string `json:"modelP"`
	RSSI      int    `json:"rssi"`
	TxPower   int    `json:"txPower"`
	Org       string `json:"org"`
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

func (trace *InMemoryTracing) RegisterNewTraceUser(ctx context.Context, UID string) (err error) {
	inMemoryLog.Infof("RegisterNewTraceUser : UID = %s", UID)
	if _, ok := trace.UIDs[UID]; !ok {
		trace.UIDs[UID] = &TraceUser{
			UID:    UID,
			Traces: make([]*TraceData, 0),
		}
	}
	return nil
}
func (trace *InMemoryTracing) GetAdminToken(ctx context.Context, uid, secret string) (token string, err error) {
	inMemoryLog.Infof("GetUploadToken : UID = %s", uid)
	if _, ok := trace.UIDs[uid]; !ok {
		return "", fmt.Errorf("%w : %s", ErrUIDNotFound, uid)
	}

	if td, ok := trace.SecretTokenMap[secret]; ok {
		return td, nil
	}
	return "", ErrTokenNotFound
}
func (trace *InMemoryTracing) SaveTraceData(ctx context.Context, UID, uploadToken string, data []*TraceData) (err error) {
	inMemoryLog.Infof("SaveTraceData : UID = %s", UID)
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
func (trace *InMemoryTracing) PurgeOldTraceData(ctx context.Context, oldestTimeStamp int64, uploadToken string) (err error) {
	inMemoryLog.Infof("PurgeOldTraceData : Purge older than unix %d", oldestTimeStamp)
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
func (trace *InMemoryTracing) GetTraceData(ctx context.Context, UID, uploadToken string) (traces []*TraceData, err error) {
	inMemoryLog.Infof("GetTraceData : UID = %s", UID)
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

type MongoDBTracing struct {
	database string
	server   string
	port     int
	user     string
	password string

	client *mongo.Client
	db     *mongo.Database
}

func NewMongoDBTracing(database, host string, port int, user, password string) ITracing {
	tracing := &MongoDBTracing{
		database: database,
		server:   host,
		port:     port,
		user:     user,
		password: password,
	}

	client, err := mongo.NewClient(options.Client().ApplyURI(tracing.getMongoURL()))
	if err != nil {
		mongoLog.Fatal(err)
		return nil
	}
	tracing.client = client
	err = client.Connect(context.TODO())
	if err != nil {
		mongoLog.Fatal(err)
		return nil
	}

	tracing.db = client.Database(tracing.database)

	return tracing
}
func (trace *MongoDBTracing) getMongoURL() string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%d", trace.user, trace.password, trace.server, trace.port)
}
func (trace *MongoDBTracing) RegisterNewTraceUser(ctx context.Context, UID string) (err error) {
	mongoLog.Infof("RegisterNewTraceUser : UID = %s", UID)
	return fmt.Errorf("not yet implemented")
}
func (trace *MongoDBTracing) GetAdminToken(ctx context.Context, UID, secret string) (token string, err error) {
	mongoLog.Infof("GetUploadToken : UID = %s", UID)
	return "", fmt.Errorf("not yet implemented")
}
func (trace *MongoDBTracing) SaveTraceData(ctx context.Context, UID, uploadToken string, data []*TraceData) (err error) {
	mongoLog.Infof("SaveTraceData : UID = %s", UID)
	return fmt.Errorf("not yet implemented")
}
func (trace *MongoDBTracing) PurgeOldTraceData(ctx context.Context, oldestTimeStamp int64, uploadToken string) (err error) {
	mongoLog.Infof("PurgeOldTraceData : Purge older than unix %d", oldestTimeStamp)
	return fmt.Errorf("not yet implemented")
}
func (trace *MongoDBTracing) GetTraceData(ctx context.Context, UID, uploadToken string) (traces []*TraceData, err error) {
	mongoLog.Infof("GetTraceData : UID = %s", UID)
	return nil, fmt.Errorf("not yet implemented")
}
