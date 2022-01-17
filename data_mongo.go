package hypertrace

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	userCollection    = "user"
	traceCollection   = "trace"
	officerCollection = "officer"
)

var (
	mongoLog = logrus.WithField("DB", "MongoDB")
)

type MongoDBTracing struct {
	database string
	server   string
	port     int
	user     string
	password string

	client *mongo.Client
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

	return tracing
}
func (trace *MongoDBTracing) getMongoURL() string {
	return fmt.Sprintf("mongodb://%s:%s@%s:%d", trace.user, trace.password, trace.server, trace.port)
}

func (trace *MongoDBTracing) getUser(ctx context.Context, UID string) (user *User, err error) {
	if len(UID) == 0 {
		return nil, ErrInvalidParameter
	}
	mongoLog.Tracef("getUser UID:%s", UID)
	userCollection := trace.client.Database(trace.database).Collection(userCollection)
	filter := bson.M{"uid": UID}
	usr := &User{}
	err = userCollection.FindOne(ctx, filter).Decode(usr)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		mongoLog.Errorf("getUser . userCollection.FindOne UID:%s got %s", UID, err.Error())
		return nil, err
	}
	return usr, nil
}

func (trace *MongoDBTracing) RegisterNewUser(ctx context.Context, UID, PIN string) (err error) {
	mongoLog.Tracef("RegisterNewUser UID:%s", UID)
	if len(UID) == 0 || len(PIN) == 0 {
		return ErrInvalidParameter
	}
	usr, err := trace.getUser(ctx, UID)
	if err != nil {
		return err
	}
	if usr == nil {
		userCollection := trace.client.Database(trace.database).Collection(userCollection)
		res, err := userCollection.InsertOne(ctx, bson.D{{"uid", UID}, {"pin", PIN}})
		if err != nil {
			mongoLog.Errorf("RegisterNewUser . userCollection.InsertOne UID:%s got %s", UID, err.Error())
			return err
		}
		mongoLog.Tracef("RegisterNewUser inserted UID:%s with id:%s", UID, res.InsertedID)
		return nil
	}
	return nil
}
func (trace *MongoDBTracing) GetHandshakePIN(ctx context.Context, UID string) (PIN string, err error) {
	mongoLog.Tracef("GetHandshakePIN UID:%s", UID)
	if len(UID) == 0 {
		return "", ErrInvalidParameter
	}
	usr, err := trace.getUser(ctx, UID)
	if err != nil {
		return "", err
	}
	if usr == nil {
		return "", ErrUIDNotFound
	}
	return usr.PIN, nil
}

func (trace *MongoDBTracing) SaveTraceData(ctx context.Context, UID, OID string, data []*TraceData) (err error) {
	if len(UID) == 0 || len(OID) == 0 {
		return ErrInvalidParameter
	}
	if data != nil && len(data) > 0 {
		mongoLog.Tracef("SaveTraceData UID:%s OID:%s, %d items", UID, OID, len(data))
		traceCollection := trace.client.Database(trace.database).Collection(traceCollection)
		documents := make([]interface{}, len(data))
		for i, d := range data {
			bd := bson.D{
				{"oid", d.OID},
				{"uid", d.UID},
				{"cuid", d.CUID},
				{"timestamp", d.Timestamp},
				{"modelC", d.ModelC},
				{"modelP", d.ModelP},
				{"rssi", d.RSSI},
				{"txPower", d.TxPower},
				{"org", d.Org},
			}
			documents[i] = bd
		}
		res, err := traceCollection.InsertMany(ctx, documents)
		if err != nil {
			mongoLog.Errorf("SaveTraceData . traceCollection.InsertMany got %s", err)
			return err
		}
		mongoLog.Tracef("SaveTraceData UID:%s OID:%s, inserted %d of %d items", UID, OID, len(res.InsertedIDs), len(data))
	}
	return nil
}
func (trace *MongoDBTracing) PurgeOldTraceData(ctx context.Context, oldestTimeStamp int64) (err error) {
	mongoLog.Tracef("PurgeOldTraceData")
	traceCollection := trace.client.Database(trace.database).Collection(traceCollection)
	filter := bson.M{
		"timestamp": bson.M{
			"$lt": oldestTimeStamp,
		}}
	res, err := traceCollection.DeleteMany(ctx, filter)
	if err != nil {
		mongoLog.Errorf("PurgeOldTraceData .  traceCollection.DeleteMany got %s", err)
		return err
	}
	mongoLog.Tracef("PurgeOldTraceData deleted %d entries", res.DeletedCount)
	return nil
}
func (trace *MongoDBTracing) GetTraceData(ctx context.Context, UID string) (traces []*TraceData, err error) {
	mongoLog.Tracef("GetTraceData UID:%s", UID)
	if len(UID) == 0 {
		return nil, ErrInvalidParameter
	}
	filter := bson.D{{"uid", UID}}
	traceCollection := trace.client.Database(trace.database).Collection(traceCollection)
	cursor, err := traceCollection.Find(ctx, filter)
	if err != nil {
		mongoLog.Errorf("GetTraceData . traceCollection.Find got %s", err)
		return nil, err
	}
	traces = make([]*TraceData, 0)
	for cursor.Next(ctx) {
		td := &TraceData{}
		err := cursor.Decode(td)
		if err != nil {
			mongoLog.Errorf("GetTraceData . cursor.Decode got %s", err.Error())
		} else {
			traces = append(traces, td)
		}
	}
	return traces, nil
}

func (trace *MongoDBTracing) getOfficer(ctx context.Context, OID string) (officer *Officer, err error) {
	if len(OID) == 0 {
		return nil, ErrInvalidParameter
	}
	mongoLog.Tracef("getOfficer OID:%s", OID)
	offCollection := trace.client.Database(trace.database).Collection(officerCollection)
	filter := bson.M{"oid": OID}
	off := &Officer{}
	err = offCollection.FindOne(ctx, filter).Decode(off)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		mongoLog.Errorf("getOfficer . offCollection.FindOne OID:%s got %s", OID, err.Error())
		return nil, err
	}
	return off, nil
}
func (trace *MongoDBTracing) RegisterNewOfficer(ctx context.Context, OID, secret string) (err error) {
	mongoLog.Tracef("RegisterNewOfficer OID:%s", OID)
	if len(OID) == 0 || len(secret) == 0 {
		return ErrInvalidParameter
	}
	off, err := trace.getOfficer(ctx, OID)
	if err != nil {
		return err
	}
	if off == nil {
		offCollection := trace.client.Database(trace.database).Collection(officerCollection)
		res, err := offCollection.InsertOne(ctx, bson.D{{"oid", OID}, {"secret", secret}})
		if err != nil {
			mongoLog.Errorf("RegisterNewOfficer . offCollection.InsertOne OID:%s got %s", OID, err.Error())
			return err
		}
		mongoLog.Tracef("RegisterNewOfficer inserted OID:%s with id:%s", OID, res.InsertedID)
		return nil
	}
	return nil
}
func (trace *MongoDBTracing) GetOfficerID(ctx context.Context, secret string) (OID string, err error) {
	mongoLog.Tracef("GetOfficerID secret:****")
	if len(secret) == 0 {
		return "", ErrInvalidParameter
	}
	offCollection := trace.client.Database(trace.database).Collection(officerCollection)
	filter := bson.M{"secret": secret}
	off := &Officer{}
	err = offCollection.FindOne(ctx, filter).Decode(off)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return "", nil
		}
		mongoLog.Errorf("getOfficer . offCollection.FindOne OID:%s got %s", OID, err.Error())
		return "", err
	}
	return off.OID, nil
}
func (trace *MongoDBTracing) DeleteOfficer(ctx context.Context, OID string) (err error) {
	mongoLog.Tracef("DeleteOfficer OID:%s", OID)
	if len(OID) == 0 {
		return ErrInvalidParameter
	}
	offCollection := trace.client.Database(trace.database).Collection(officerCollection)
	filter := bson.M{"oid": OID}
	_, err = offCollection.DeleteOne(ctx, filter)
	if err != nil {
		mongoLog.Errorf("DeleteOfficer OID:%s got %s", OID, err.Error())
		return err
	}
	mongoLog.Tracef("DeleteOfficer OID:%s deleted", OID)
	return nil
}
