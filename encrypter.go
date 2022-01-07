package hypertrace

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func encode(cyperText, iv []byte) (encoded []byte, err error) {
	buff := &bytes.Buffer{}
	err = buff.WriteByte(byte(len(cyperText)))
	if err != nil {
		return nil, fmt.Errorf("%w : encode error", err)
	}
	_, err = buff.Write(cyperText)
	if err != nil {
		return nil, fmt.Errorf("%w : encode error", err)
	}

	err = buff.WriteByte(byte(len(iv)))
	if err != nil {
		return nil, fmt.Errorf("%w : encode error", err)
	}
	_, err = buff.Write(iv)
	if err != nil {
		return nil, fmt.Errorf("%w : encode error", err)
	}

	return buff.Bytes(), nil
}

func decode(encoded []byte) (cyperText, iv []byte, err error) {
	var cyperData, ivData []byte
	buff := bytes.NewBuffer(encoded)

	cypherLen, err := buff.ReadByte()
	if err != nil {
		return nil, nil, fmt.Errorf("%w : decode error during reading 1 byte from buffer to find out cypher len", err)
	}
	cyperData = make([]byte, cypherLen)
	_, err = buff.Read(cyperData)
	if err != nil {
		return nil, nil, fmt.Errorf("%w : decode error during reading %d bytes of cypher data ", err, cypherLen)
	}

	ivLen, err := buff.ReadByte()
	if err != nil {
		return nil, nil, fmt.Errorf("%w : decode error during reading 1 byte from buffer to find out IV len", err)
	}

	ivData = make([]byte, ivLen)
	_, err = buff.Read(ivData)
	if err != nil {
		return nil, nil, fmt.Errorf("%w : decode error during reading %d bytes of iv data", err, ivLen)
	}

	return cyperData, ivData, nil
}

func encrypt(data, key []byte) (cypherText, iv []byte, err error) {
	if len(key) != 32 {
		return nil, nil, fmt.Errorf("invalid key size %d, we expect 32", len(key))
	}
	iv = make([]byte, 12)
	_, err = rand.Read(iv)
	if err != nil {
		return nil, nil, fmt.Errorf("%w : encrypt error", err)
	}

	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("%w : encrypt error", err)
	}
	encrypter, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, nil, fmt.Errorf("%w : encrypt error", err)
	}

	cypherText = encrypter.Seal(nil, iv, data, nil)

	return cypherText, iv, nil
}

func decrypt(cypherText, iv, key []byte) (data []byte, err error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("decrypt error : invalid key size %d, we expect 32", len(key))
	}
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w : decrypt error", err)
	}
	decryptor, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("%w : decrypt error", err)
	}
	data, err = decryptor.Open(nil, iv, cypherText, nil)
	if err != nil {
		return nil, fmt.Errorf("%w : decryptor.Open error", err)
	}
	return data, err
}

func encryptAndEncode(data, key []byte) (crypted string, err error) {
	cypherText, iv, err := encrypt(data, key)
	if err != nil {
		return "", fmt.Errorf("%w : encrypt-encode error", err)
	}
	encoded, err := encode(cypherText, iv)
	if err != nil {
		return "", fmt.Errorf("%w : encrypt-encode error", err)
	}
	return B64Encode(encoded), nil
}

func decodeAndDecrypt(crypted string, key []byte) (data []byte, err error) {
	cypher, iv, err := decode(B64Decode(crypted))
	if err != nil {
		return nil, fmt.Errorf("%w : decode error when decoding base64 cryptext", err)
	}
	data, err = decrypt(cypher, iv, key)
	if err != nil {
		return nil, fmt.Errorf("%w : decrypt error", err)
	}
	return data, err
}

type SymetricKey struct {
	Key string `json:"key"`
	Iv  string `json:"iv"`
}

func B64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func B64Decode(b64 string) []byte {
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		panic(err.Error())
	}
	return decoded
}
