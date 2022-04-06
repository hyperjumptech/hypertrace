package hypertrace

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hyperjumptech/hypertrace/static"
	"github.com/sirupsen/logrus"
)

const (
	UID_SIZE     = 21
	TIME_SIZE    = 4
	TEMPID_SIZE  = UID_SIZE + TIME_SIZE*2
	IV_SIZE      = 16
	AUTHTAG_SIZE = 16
)

var (
	ErrInvalidTempIDLength = fmt.Errorf("invalid temporary id length")
	Tracing                ITracing
	Forwarder              IForwarder
	ENCRYPTIONKEY          string
	ValidPeriod            uint32
	TempIDAmount           int
)

func init() {
	ENCRYPTIONKEY = ConfigGet("tempid.crypt.key")
	ValidPeriod = uint32(ConfigGetInt("tempid.valid.period.hour"))
	TempIDAmount = ConfigGetInt("tempid.count")

	Forwarder = &StdOutForwarder{}
}

func InitTracing() {
	if Tracing == nil {
		if ConfigGet("database") == "mongodb" {
			logrus.Warnf("Database using MongoDB")
			Tracing = NewMongoDBTracing(ConfigGet("mongo.database"), ConfigGet("mongo.host"), ConfigGetInt("mongo.port"), ConfigGet("mongo.user"), ConfigGet("mongo.password"))
		} else {
			logrus.Warnf("Database using InMemory. Next server restart will clear all data.")
			Tracing = NewInMemoryTracing()
		}
	}
}

func registerUid(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	if len(uid) != UID_SIZE {
		logrus.Errorf("registerUid: uid %s < %d characters", uid, UID_SIZE)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"status\":\"FAIL\""))
		return
	}
	pin := r.URL.Query().Get("pin")
	if len(pin) == 0 {
		logrus.Errorf("registerUid: empty pin")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"status\":\"FAIL\""))
		return
	}
	secret := r.URL.Query().Get("secret")
	_, err := Tracing.GetOfficerID(r.Context(), secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{\"status\":\"FAIL\""))
		return
	}

	err = Tracing.RegisterNewUser(r.Context(), uid, pin)
	if err != nil {
		logrus.Errorf("registerUid: got %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"status\":\"SUCCESS\"}"))
}

func getHandshakePin(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	if len(uid) != UID_SIZE {
		logrus.Errorf("getHandshakePin: uid %s < %d characters", uid, UID_SIZE)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"status\":\"FAIL\""))
		return
	}
	logrus.Infof("getHandshakePin: uid %s", uid)

	pin, err := Tracing.GetHandshakePIN(r.Context(), uid)
	if err != nil {
		if errors.Is(err, ErrUIDNotFound) {
			w.Header().Add("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("uid specified not found"))
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{\"status\":\"SUCCESS\", \"pin\":\"%s\"}", pin)))
}

func registerOfficer(w http.ResponseWriter, r *http.Request) {
	adminpassword := r.URL.Query().Get("pass")
	oid := r.URL.Query().Get("oid")
	secret := r.URL.Query().Get("secret")

	if adminpassword != ConfigGet("adminpassword") {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
		return
	}

	if len(oid) == 0 || len(secret) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing oid or secret"))
		return
	}

	err := Tracing.RegisterNewOfficer(r.Context(), oid, secret)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{\"status\":\"SUCCESS\"}")))
}

func deleteOfficer(w http.ResponseWriter, r *http.Request) {
	adminpassword := r.URL.Query().Get("pass")
	oid := r.URL.Query().Get("oid")

	if adminpassword != ConfigGet("adminpassword") {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("unauthorized"))
		return
	}

	if len(oid) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing oid or secret"))
		return
	}

	err := Tracing.DeleteOfficer(r.Context(), oid)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{\"status\":\"SUCCESS\"}")))
}

type TempIDResponse struct {
	Status      string    `json:"status"`
	TempIDs     []*TempID `json:"tempIDs"`
	RefreshTime uint32    `json:"refreshTime"`
}

func purgeTracing(w http.ResponseWriter, r *http.Request) {
	secret := r.URL.Query().Get("secret")
	sAgeHour := r.URL.Query().Get("ageHour")
	if len(secret) == 0 || len(sAgeHour) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing secret or ageHour parameter"))
		return
	}
	ageHour, err := strconv.ParseInt(sAgeHour, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid timestamp format"))
		return
	}

	age := time.Duration(ageHour) * time.Hour
	oldest := time.Now().Add(-age)

	_, err = Tracing.GetOfficerID(r.Context(), secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid secret format"))
		return
	}

	err = Tracing.PurgeOldTraceData(r.Context(), oldest.Unix())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid uploadToken format"))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"status\":\"SUCCESS\"}"))
	return
}

func getTempIDs(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")

	tempIds, err := GenerateTempIDs(uid)
	if err != nil && err == ErrInvalidTempIDLength {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"status\":\"FAIL\"}"))
		return
	}

	resp := &TempIDResponse{
		Status:      "SUCCESS",
		TempIDs:     tempIds,
		RefreshTime: uint32(time.Now().Unix() + 3600*24),
	}
	respJson, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respJson)
}

func getUploadToken(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	secret := r.URL.Query().Get("secret")

	if len(uid) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing 'uid' param"))
		return
	}
	if len(secret) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing 'data' param"))
		return
	}

	oid, err := Tracing.GetOfficerID(r.Context(), secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("secret not valid"))
		return
	}

	ut := NewUploadToken(uid, oid, 1)
	tok, err := ut.ToToken([]byte(ENCRYPTIONKEY))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{\"status\":\"SUCCESS\", \"token\":\"%s\"}", tok)))
	return
}

func uploadData(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		logrus.Error(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	upload := &DataUpload{}
	err = json.Unmarshal(bodyBytes, upload)
	if err != nil {
		logrus.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	// validate the token
	ut, err := NewUploadTokenFromString(upload.UploadToken, []byte(ENCRYPTIONKEY))
	if err != nil {
		logrus.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	if !ut.IsValid() {
		logrus.Error(err.Error())
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("upload token expired"))
		return
	}

	traces := make([]*TraceData, 0)

	for _, tr := range upload.Traces {
		TempID := tr.Message
		uid, start, exp, err := GetTempIDData([]byte(ENCRYPTIONKEY), TempID)
		if err != nil {
			logrus.Error(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		logrus.Tracef("todo to check if tr.Timestamp %d is between %d and %d", tr.Timestamp, start, exp)
		// todo Make sure the tr.Timestamp is within start and exp

		td := &TraceData{
			CUID:      uid,
			Timestamp: tr.Timestamp,
			ModelC:    tr.ModelC,
			ModelP:    tr.ModelP,
			RSSI:      tr.RSSI,
			TxPower:   tr.TxPower,
			Org:       tr.Org,
		}

		traces = append(traces, td)
	}

	err = Tracing.SaveTraceData(r.Context(), upload.UID, ut.OID, traces)
	if err != nil && (errors.Is(err, ErrUIDNotFound) || errors.Is(err, ErrTokenNotFound)) {
		logrus.Error(err.Error())
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(fmt.Sprintf("uid or upload token not found. got %s", err.Error())))
		return
	}

	if err != nil {
		logrus.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	err = Forwarder.ForwardTraceData(upload.UID, traces)
	if err != nil {
		logrus.Errorf("forwarder error. got %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{\"status\":\"SUCCESS\"}"))
}

type TracingResponse struct {
	Status  string       `json:"status"`
	Tracing []*TraceData `json:"trace"`
}

func getTracing(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	secret := r.URL.Query().Get("secret")
	_, err := Tracing.GetOfficerID(r.Context(), secret)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("invalid secret"))
		return
	}
	tdata, err := Tracing.GetTraceData(r.Context(), uid)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	} else {
		tr := &TracingResponse{
			Status:  "SUCCESS",
			Tracing: tdata,
		}
		respBytes, _ := json.Marshal(tr)
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(respBytes)
	}
}

func GenerateTempIDs(uid string) (tempIds []*TempID, err error) {
	if len(uid) < UID_SIZE {
		return nil, ErrInvalidTempIDLength
	}
	tempIds = make([]*TempID, TempIDAmount)
	for i := 0; i < len(tempIds); i++ {
		tempId, err := generateTempId([]byte(ENCRYPTIONKEY), uid, uint32(i))
		if err != nil {
			logrus.Errorf(err.Error())
		}
		tempIds[i] = tempId
	}
	return tempIds, nil
}

type TempID struct {
	TempID     string `json:"tempID"`
	StartTime  uint32 `json:"startTime"`
	ExpiryTime uint32 `json:"expiryTime"`
}

func (tid *TempID) IsValid(key []byte, forTime time.Time) bool {
	_, err := decodeAndDecrypt(tid.TempID, key)
	if err != nil {
		return false
	}
	return true
}

func GetTempIDData(key []byte, tempid string) (UID string, start, expiry int32, err error) {
	data, err := decodeAndDecrypt(tempid, key)
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w : error processing tempID %s for decodingAndDecrypt process", err, tempid)
	}
	buff := bytes.NewBuffer(data)

	uidBytes := make([]byte, UID_SIZE)
	startBytes := make([]byte, 4)
	expiryBytes := make([]byte, 4)

	_, err = io.ReadFull(buff, uidBytes)
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w : error processing tempID %s for extracting UID bytes", err, tempid)
	}
	_, err = io.ReadFull(buff, startBytes)
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w : error processing tempID %s for extracting Start bytes", err, tempid)
	}
	_, err = io.ReadFull(buff, expiryBytes)
	if err != nil {
		return "", 0, 0, fmt.Errorf("%w : error processing tempID %s for extracting Expire bytes", err, tempid)
	}

	start = int32(binary.BigEndian.Uint32(startBytes))
	expiry = int32(binary.BigEndian.Uint32(expiryBytes))

	return string(uidBytes), start, expiry, nil
}

func generateTempId(key []byte, uid string, i uint32) (*TempID, error) {
	now := uint32(time.Now().Unix())
	start := now + 3600*ValidPeriod*i - 60
	expiry := start + 3600*ValidPeriod

	buff := &bytes.Buffer{}
	buff.Write([]byte(uid))

	startBytes := make([]byte, 4)
	expiryBytes := make([]byte, 4)

	binary.BigEndian.PutUint32(startBytes, start)
	binary.BigEndian.PutUint32(expiryBytes, expiry)

	buff.Write(startBytes)
	buff.Write(expiryBytes)

	val, err := encryptAndEncode(buff.Bytes(), key)

	return &TempID{
		TempID:     val,
		StartTime:  start,
		ExpiryTime: expiry,
	}, err
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func StaticMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		if !strings.HasPrefix(req.URL.Path, "/docs") {
			next.ServeHTTP(res, req)
		} else {
			if req.Method != http.MethodGet {
				res.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if req.URL.Path == "/docs" || req.URL.Path == "/docs/" {
				res.Header().Set("Location", "/docs/index.html")
				res.WriteHeader(http.StatusMovedPermanently)
				return
			} else if strings.HasSuffix(req.URL.Path, "/") {
				res.Header().Set("Location", req.URL.Path+"index.html")
				res.WriteHeader(http.StatusMovedPermanently)
				return
			}
			filePath := strings.ReplaceAll(req.URL.Path, "/docs/", "api/")
			dirFilePath := "[DIR]" + filePath
			paths := static.GetPathTree("api")
			for _, path := range paths {
				if path == dirFilePath {
					res.Header().Set("Location", req.URL.Path+"/index.html")
					res.WriteHeader(http.StatusMovedPermanently)
					return
				}
				if path == filePath {
					fdata, err := static.GetFile(filePath)
					if err != nil {
						res.WriteHeader(http.StatusInternalServerError)
						res.Write([]byte(err.Error()))
						return
					}
					res.Header().Set("Content-Type", fdata.ContentType)
					res.WriteHeader(http.StatusOK)
					res.Write(fdata.Bytes)
					return
				}
			}
			res.WriteHeader(http.StatusNotFound)
		}
	})
}
