package crypto

import "testing"

func TestEncryptDecrypt(t *testing.T) {
	secret := "test-secret-key"
	plain := `{"api_token":"abc"}`
	enc, err := Encrypt(secret, plain)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := Decrypt(secret, enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != plain {
		t.Fatalf("got %q want %q", dec, plain)
	}
}
