# OpenTrace Server - Hyperjump Golang Implementation

## 1. getHandshakePin

### 1.1 Request

| Request | Info    |
|:--------|:--------|
| Method | GET    |
| Path   | `/getHandshakePin?uid=<uid>` |

**Query Strings**

| Key     | Description    |
|:--------|:--------|
| uid     | The user's identification code. This code should be usable by implementor to identify the actual user. The uid it self should be meaningless to outside the organization, thus it should not store phone number, email or informations alike. **uid must be 21 character**|

**Example**

```shell
GET http://server/getHandshakePin?uid=jlhk3wl3lglwjbiopcisa
```

### 1.2 Response

**Response Body**

```json
{
	"status": "SUCCESS",
	"pin": "123456"
}
```

## 2. getTempIDs

### 2.1 Request

### 2.1 Response

## 3. getUploadToken

### 3.1 Request

| Request | Info    |
|:--------|:--------|
| Method | GET    |
| Path   | `/getUploadToken?uid=<uid>&data=<data>` |


**Query Strings**

| Key     | Description    |
|:--------|:--------|
| uid     | The user's identification code. This code should be usable by implementor to identify the actual user. The uid it self should be meaningless to outside the organization, thus it should not store phone number, email or informations alike. **uid must be 21 character**|
| data | This is actually a secret data that should be supplied by the tracing authority, used for uploading the trace result into the authorities's database |


### 3.2 Response

```json
{
	"status": "SUCCESS",
	"token": "UAMnfvgwsXW96kZu"
}
```

## 4. uploadData

### 4.1 Request

| Request | Info    |
|:--------|:--------|
| Method | POST    |
| Path   | `/uploadData` |

```json
{
	"uid": "123456789012345678901",
	"uploadToken": "",
	"traces": [
		{
			"timestamp":1641378852,
			"msg":"LQdrDdfT7kaipalO846vy+wUOQVeW5ct2N/pVOMWxi/7y27VWqsIW9ggsaIqHQyK37WNY+nkSmV5L7w=",
			"modelC":"Samsung S4",
			"modelP":"IOS14",
			"rssi": 6,
			"txPower": 34,
			"org":"hyperjump"
		},
		{
			"timestamp":1641378852,
			"msg":"LUhfuCwd6y6cpKt5gb+AEFwhiRqasfmF4nlZuLYChhAGgicH/KLAEIiLZmXxcgxNGU9ySVxstsXeUyg=",
			"modelC":"Samsung S4",
			"modelP":"IOS14",
			"rssi": 6,
			"txPower": 34,
			"org":"hyperjump"
		} 
	]
}
```


### 4.2 Response

```json
{
	"status": "SUCCESS"
}
```

## 5. getTracing

### 5.1 Request

| Request | Info    |
|:--------|:--------|
| Method | GET    |
| Path   | `/getTracing?uid=<uid>&uploadToken=<uploadToken>` |


**Query Strings**

| Key     | Description    |
|:--------|:--------|
| uid     | The user's identification code. This code should be usable by implementor to identify the actual user. The uid it self should be meaningless to outside the organization, thus it should not store phone number, email or informations alike. **uid must be 21 character**|
| uploadToken | The upload token obtained from calling `getUploadToken` |


### 5.2 Response

```json
{
  "status": "SUCCESS",
  "trace": [
    {
      "ContactUID": "123456789012345678901",
      "Timestamp": 1641378852,
      "ModelC": "Samsung S4",
      "ModelP": "IOS14",
      "RSSI": 6,
      "TxPower": 34,
      "Org": "hyperjump"
    },
    {
      "ContactUID": "123456789012345678901",
      "Timestamp": 1641378852,
      "ModelC": "Samsung S4",
      "ModelP": "IOS14",
      "RSSI": 6,
      "TxPower": 34,
      "Org": "hyperjump"
    }
  ]
}
```

## 6. purgeTracing

### 6.1 Request

| Request | Info    |
|:--------|:--------|
| Method | GET    |
| Path   | `/purgeTracing?uploadToken=<uploadToken>&ageHour=<ageHour>` |

**Query Strings**

| Key     | Description    |
|:--------|:--------|
| uploadToken | The upload token obtained from calling `getUploadToken` |
| ageHour | ALL trace data belong to ALL uid, older than this hour will be purged. Setting this to 0 will clear all trace data |


### 6.2 Response

```json
{
	"status": "SUCCESS"
}
```