package hypertrace

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	ValidPeriod = 5
	UID_SIZE    = 21
	TIME_SIZE   = 4
	// 21 bytes for UID, 4 bytes each for creation and expiry timestamp
	TEMPID_SIZE   = UID_SIZE + TIME_SIZE*2
	IV_SIZE       = 16
	AUTHTAG_SIZE  = 16
	ENCRYPTIONKEY = "tH1Sis4nEncryPt10nKeydOn0tsHar3!"
)

var (
	ErrInvalidTempIDLength = fmt.Errorf("invalid temporary id length")
	Tracing                ITracing
)

func init() {
	Tracing = NewInMemoryTracing()
}

func getHandshakePin(w http.ResponseWriter, r *http.Request) {
	uid := r.URL.Query().Get("uid")
	if len(uid) != UID_SIZE {
		logrus.Errorf("getHandshakePin: uid %s < %d characters", uid, UID_SIZE)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{\"status\":\"FAIL\""))
		return
	}

	_ = Tracing.RegisterNewTraceUser(uid)
	logrus.Infof("getHandshakePin: uid %s", uid)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{\"status\":\"SUCCESS\", \"pin\":\"%s\"}", generatePIN(uid))))
}

type TempIDResponse struct {
	Status      string    `json:"status"`
	TempIDs     []*TempID `json:"tempIDs"`
	RefreshTime uint32    `json:"refreshTime"`
}

func purgeTracing(w http.ResponseWriter, r *http.Request) {
	uploadToken := r.URL.Query().Get("uploadToken")
	sAgeHour := r.URL.Query().Get("ageHour")
	if len(uploadToken) == 0 || len(sAgeHour) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing uploadToken or ageHour parameter"))
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

	err = Tracing.PurgeOldTraceData(oldest.Unix(), uploadToken)
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
	data := r.URL.Query().Get("data")

	if len(uid) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing 'uid' param"))
		return
	}
	if len(data) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing 'data' param"))
		return
	}

	token, err := Tracing.GetUploadToken(uid, data)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Supplied uid or data not found"))
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("{\"status\":\"SUCCESS\", \"token\":\"%s\"}", token)))
	return
}

func uploadData(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	upload := &DataUpload{}
	err = json.Unmarshal(bodyBytes, upload)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	traces := make([]*TraceData, 0)

	for _, tr := range upload.Traces {
		TempID := tr.Message
		uid, start, exp, err := GetTempIDData([]byte(ENCRYPTIONKEY), TempID)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		logrus.Tracef("todo to check if tr.Timestamp %d is between %d and %d", tr.Timestamp, start, exp)
		// todo Make sure the tr.Timestamp is within start and exp

		td := &TraceData{
			ContactUID: uid,
			Timestamp:  tr.Timestamp,
			ModelC:     tr.ModelC,
			ModelP:     tr.ModelP,
			RSSI:       tr.RSSI,
			TxPower:    tr.TxPower,
			Org:        tr.Org,
		}

		traces = append(traces, td)
	}

	err = Tracing.SaveTraceData(upload.UID, upload.UploadToken, traces)
	if err != nil && (errors.Is(err, ErrUIDNotFound) || errors.Is(err, ErrTokenNotFound)) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("uid or upload token not found"))
		return
	}
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
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
	uploadToken := r.URL.Query().Get("uploadToken")
	tdata, err := Tracing.GetTraceData(uid, uploadToken)
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

func generatePIN(uid string) string {
	return strings.ToUpper(uid)[0:6]
}

func GenerateTempIDs(uid string) (tempIds []*TempID, err error) {
	if len(uid) < UID_SIZE {
		return nil, ErrInvalidTempIDLength
	}
	tempIds = make([]*TempID, 100)
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
		return "", 0, 0, err
	}
	buff := bytes.NewBuffer(data)

	uidBytes := make([]byte, UID_SIZE)
	startBytes := make([]byte, 4)
	expiryBytes := make([]byte, 4)

	_, err = io.ReadFull(buff, uidBytes)
	if err != nil {
		return "", 0, 0, err
	}
	_, err = io.ReadFull(buff, startBytes)
	if err != nil {
		return "", 0, 0, err
	}
	_, err = io.ReadFull(buff, expiryBytes)
	if err != nil {
		return "", 0, 0, err
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
