package hypertrace

import "testing"

func TestEncryptDecrypt(t *testing.T) {
	dataString := "thequickbrownfoxjumpsoveralazydogthequickbrownfoxjumpsoveralazydogthequickbrownfoxjumpsoveralazydogthequickbrownfoxjumpsoveralazydog"
	key := "thisistheencryptionkey0123456789"

	crypted, err := encryptAndEncode([]byte(dataString), []byte(key))
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}

	data, err := decodeAndDecrypt(crypted, []byte(key))
	if err != nil {
		t.Errorf("error from decodeAndDecrypt : %v", err.Error())
		t.FailNow()
	}

	if dataString != string(data) {
		t.Error("Not equal")
		t.FailNow()
	}
}
