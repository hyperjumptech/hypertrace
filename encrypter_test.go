package hypertrace

import "testing"

func TestDecrypt(t *testing.T) {
	data := "LZYJvvdD2VI7rF+pv5Bm8bkg0ZDvQe/ad5lu6T5YWdwEreVLrCLhUtXjm6hE5AzqmEmeGP8Vdlbnt+c="
	dataDec, err := decodeAndDecrypt(data, []byte(ENCRYPTIONKEY))
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(dataDec)
}

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
