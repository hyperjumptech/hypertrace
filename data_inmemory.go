package hypertrace

import (
	"context"
	"github.com/sirupsen/logrus"
)

var (
	inMemoryLog = logrus.WithField("DB", "InMemory")
)

func NewInMemoryTracing() ITracing {
	tracing := &InMemoryTracing{
		Users:      make(map[string]*User),
		Officers:   make(map[string]*Officer),
		TraceDatas: make([]*TraceData, 0),
	}
	_ = tracing.RegisterNewOfficer(context.Background(), "officer1", "secret1")
	_ = tracing.RegisterNewOfficer(context.Background(), "officer2", "secret2")
	_ = tracing.RegisterNewOfficer(context.Background(), "officer3", "secret3")
	_ = tracing.RegisterNewOfficer(context.Background(), "officer4", "secret4")
	_ = tracing.RegisterNewOfficer(context.Background(), "officer5", "secret5")
	return tracing
}

type InMemoryTracing struct {
	Users      map[string]*User
	Officers   map[string]*Officer
	TraceDatas []*TraceData
}

func (trace *InMemoryTracing) RegisterNewUser(ctx context.Context, UID, PIN string) (err error) {
	inMemoryLog.Tracef("RegisterNewUser UID:%s", UID)
	if len(UID) == 0 || len(PIN) == 0 {
		return ErrInvalidParameter
	}
	trace.Users[UID] = &User{
		UID: UID,
		PIN: PIN,
	}
	return nil
}
func (trace *InMemoryTracing) GetHandshakePIN(ctx context.Context, UID string) (PIN string, err error) {
	inMemoryLog.Tracef("GetHandshakePIN UID:%s", UID)
	if tu, ok := trace.Users[UID]; ok {
		return tu.PIN, nil
	}
	return "", ErrUIDNotFound
}

func (trace *InMemoryTracing) SaveTraceData(ctx context.Context, UID, OID string, data []*TraceData) (err error) {
	inMemoryLog.Tracef("SaveTraceData UID:%s OID:%s", UID, OID)
	for _, tdata := range data {
		tdata.UID = UID
		tdata.OID = OID
	}
	trace.TraceDatas = append(trace.TraceDatas, data...)
	return nil
}
func (trace *InMemoryTracing) PurgeOldTraceData(ctx context.Context, oldestTimeStamp int64) (err error) {
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
func (trace *InMemoryTracing) GetTraceData(ctx context.Context, UID string) (traces []*TraceData, err error) {
	inMemoryLog.Tracef("GetTraceData UID:%s", UID)
	newTraceData := make([]*TraceData, 0)
	for _, td := range trace.TraceDatas {
		if td.UID == UID {
			newTraceData = append(newTraceData, td)
		}
	}
	return newTraceData, nil
}

func (trace *InMemoryTracing) RegisterNewOfficer(ctx context.Context, OID, secret string) (err error) {
	inMemoryLog.Tracef("RegisterNewOfficer OID:%s", OID)
	trace.Officers[OID] = &Officer{
		OID:    OID,
		Secret: secret,
	}
	return nil
}
func (trace *InMemoryTracing) GetOfficerID(ctx context.Context, secret string) (OID string, err error) {
	inMemoryLog.Tracef("GetOfficerID secret:%s", secret)
	for oid, off := range trace.Officers {
		if off.Secret == secret {
			return oid, nil
		}
	}
	return "", ErrSecretNotValid
}
func (trace *InMemoryTracing) DeleteOfficer(ctx context.Context, OID string) (err error) {
	inMemoryLog.Tracef("DeleteOfficer OID:%s", OID)
	delete(trace.Officers, OID)
	return nil
}
