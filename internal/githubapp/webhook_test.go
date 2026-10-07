package githubapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerify(t *testing.T) {
	body := []byte(`{"zen":"nice"}`)
	mac := hmac.New(sha256.New, []byte("s3cret"))
	mac.Write(body)
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if err := Verify("s3cret", sig, body); err != nil {
		t.Fatal(err)
	}
	if err := Verify("s3cret", "sha256=deadbeef", body); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestHandleDuplicate(t *testing.T) {
	dir := t.TempDir()
	body := []byte(`{"repository":{"full_name":"acme/widget"}}`)
	a, err := Handle(dir, "push", "deliv-1", body)
	if err != nil {
		t.Fatal(err)
	}
	if a.Duplicate || a.Repo != "acme/widget" {
		t.Fatalf("%+v", a)
	}
	b, err := Handle(dir, "push", "deliv-1", body)
	if err != nil {
		t.Fatal(err)
	}
	if !b.Duplicate {
		t.Fatal("expected duplicate delivery to be ignored")
	}
}
