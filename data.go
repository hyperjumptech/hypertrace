package hypertrace

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"time"
)

const (
	UploadTokenLength = 16
)

var (
	ErrRegisterUIDError = fmt.Errorf("uid registration error")
	ErrUIDNotFound      = fmt.Errorf("uid not found")
	ErrTokenNotFound    = fmt.Errorf("token not found")
	ErrSecretNotValid = fmt.Errorf("secret not valid")

	inMemoryLog         = logrus.WithField("DB", "InMemory")
	mongoLog            = logrus.WithField("DB", "MongoDB")

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
	OID string `json:"oid"`
	UID string `json:"uid"`
	ValidFrom int64 `json:"nbf"`
	ValidUntil int64 `json:"exp"`
}

func (ut *UploadToken) IsValid() bool {
	n := time.Now().Unix()
	return n > ut.ValidFrom && n < ut.ValidUntil
}

func (ut *UploadToken) ToToken(key []byte) (token string, err error)  {
	utBytes, err  := json.Marshal(ut)
	if err != nil {
		return "", err
	}
	return encryptAndEncode(utBytes, key)
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
		Users:           make(map[string]*User),
		Officers: make(map[string]*Officer),
		TraceDatas: make([]*TraceData,0),
	}
	_ = tracing.RegisterNewOfficer(context.Background(), "officer1", "secret1")
	_ = tracing.RegisterNewOfficer(context.Background(), "officer2", "secret2")
	_ = tracing.RegisterNewOfficer(context.Background(), "officer3", "secret3")
	_ = tracing.RegisterNewOfficer(context.Background(), "officer4", "secret4")
	_ = tracing.RegisterNewOfficer(context.Background(), "officer5", "secret5")
	return tracing
}

type InMemoryTracing struct {
	Users map[string]*User
	Officers map[string]*Officer
	TraceDatas []*TraceData
}

func (trace *InMemoryTracing) RegisterNewUser(ctx context.Context, UID, PIN string) (err error) {
	inMemoryLog.Tracef("RegisterNewUser UID:%s", UID)
	trace.Users[UID] = &User{
		UID: UID,
		PIN: PIN,
	}
	return nil
}
func (trace *InMemoryTracing) GetHandshakePIN(ctx context.Context, UID string) (PIN string, err error){
	inMemoryLog.Tracef("GetHandshakePIN UID:%s", UID)
	if tu, ok := trace.Users[UID]; ok {
		return tu.PIN, nil
	}
	return "", ErrUIDNotFound
}


func (trace *InMemoryTracing) SaveTraceData(ctx context.Context, UID, OID string, data []*TraceData) (err error){
	inMemoryLog.Tracef("SaveTraceData UID:%s OID:%s", UID, OID)
	for _,tdata := range data {
		tdata.UID = UID
		tdata.OID = OID
	}
	trace.TraceDatas = append(trace.TraceDatas, data... )
	return nil
}
func (trace *InMemoryTracing) PurgeOldTraceData(ctx context.Context, oldestTimeStamp int64) (err error){
	inMemoryLog.Tracef("PurgeOldTraceData")
	newTraceData := make([]*TraceData, 0)
	for _, td := range trace.TraceDatas {
		if td.Timestamp >= oldestTimeStamp {
			newTraceData = append(newTraceData, td)
		}
	}
	trace.TraceDatas = newTraceData
	return nil
}
func (trace *InMemoryTracing) GetTraceData(ctx context.Context, UID string) (traces []*TraceData, err error){
	inMemoryLog.Tracef("GetTraceData UID:%s", UID)
	newTraceData := make([]*TraceData, 0)
	for _, td := range trace.TraceDatas {
		if td.UID == UID {
			newTraceData = append(newTraceData, td)
		}
	}
	return newTraceData, nil
}

func (trace *InMemoryTracing) RegisterNewOfficer(ctx context.Context, OID, secret string) (err error){
	inMemoryLog.Tracef("RegisterNewOfficer OID:%s", OID)
	trace.Officers[OID] = &Officer{
		OID:    OID,
		Secret: secret,
	}
	return nil
}
func (trace *InMemoryTracing) GetOfficerID(ctx context.Context, secret string) (OID string, err error){
	inMemoryLog.Tracef("GetOfficerID secret:%s", secret)
	for oid, off := range trace.Officers {
		if off.Secret == secret {
			return oid, nil
		}
	}
	return "", ErrSecretNotValid
}
func (trace *InMemoryTracing) DeleteOfficer(ctx context.Context, OID string) (err error){
	inMemoryLog.Tracef("DeleteOfficer OID:%s", OID)
	delete(trace.Officers, OID)
	return nil
}

/**
MongoDB Implementations
 */

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


func (trace *MongoDBTracing) RegisterNewUser(ctx context.Context, UID, PIN string) (err error) {
	mongoLog.Tracef("RegisterNewUser UID:%s", UID)
	return nil
}
func (trace *MongoDBTracing) GetHandshakePIN(ctx context.Context, UID string) (PIN string, err error){
	mongoLog.Tracef("GetHandshakePIN UID:%s", UID)
	return "", ErrUIDNotFound
}


func (trace *MongoDBTracing) SaveTraceData(ctx context.Context, UID, OID string, data []*TraceData) (err error){
	mongoLog.Tracef("SaveTraceData UID:%s OID:%s", UID, OID)
	return nil
}
func (trace *MongoDBTracing) PurgeOldTraceData(ctx context.Context, oldestTimeStamp int64) (err error){
	mongoLog.Tracef("PurgeOldTraceData")
	return nil
}
func (trace *MongoDBTracing) GetTraceData(ctx context.Context, UID string) (traces []*TraceData, err error){
	mongoLog.Tracef("GetTraceData UID:%s", UID)
	return nil, nil
}

func (trace *MongoDBTracing) RegisterNewOfficer(ctx context.Context, OID, secret string) (err error){
	mongoLog.Tracef("RegisterNewOfficer OID:%s", OID)
	return nil
}
func (trace *MongoDBTracing) GetOfficerID(ctx context.Context, secret string) (OID string, err error){
	mongoLog.Tracef("GetOfficerID secret:%s", secret)
	return "", ErrSecretNotValid
}
func (trace *MongoDBTracing) DeleteOfficer(ctx context.Context, OID string) (err error){
	mongoLog.Tracef("DeleteOfficer OID:%s", OID)
	return nil
}
