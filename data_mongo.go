package hypertrace

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	mongoLog            = logrus.WithField("DB", "MongoDB")
)

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

