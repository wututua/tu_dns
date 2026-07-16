package record

import "testing"

func TestValidateRecord(t *testing.T) {
	if err := validateRecord("A", "1.2.3.4"); err != nil {
		t.Fatal(err)
	}
	if err := validateRecord("A", "bad"); err == nil {
		t.Fatal("expected error")
	}
	if err := validateRecord("CNAME", "a.example.com"); err != nil {
		t.Fatal(err)
	}
}
