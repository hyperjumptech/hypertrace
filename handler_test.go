package hypertrace

import "testing"

func TestGetTempIDData(t *testing.T) {
	tempIds := []string{
		"LZYJvvdD2VI7rF+pv5Bm8bkg0ZDvQe/ad5lu6T5YWdwEreVLrCLhUtXjm6hE5AzqmEmeGP8Vdlbnt+c=",
		"LbDQGAjIEKtM3r/e9XD9ScOZLw4i2JyFH9M3SQHLmDJZr1KBJwaub/3NMfjDegzhN45P33lsoUVIywg=",
	}
	for _, tid := range tempIds {
		_, _, _, err := GetTempIDData([]byte(ENCRYPTIONKEY), tid)
		if err != nil {
			t.Errorf("%s - %s", tid, err.Error())
			t.FailNow()
		}
	}
}
