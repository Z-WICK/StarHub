package security

import "testing"

func TestTokenCipherEncryptDecrypt(t *testing.T) {
	cipher, err := NewTokenCipher("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("expected cipher init success, got error: %v", err)
	}

	plain := "ghp_very_secret_token"
	encrypted, err := cipher.Encrypt(plain)
	if err != nil {
		t.Fatalf("expected encrypt success, got error: %v", err)
	}
	if encrypted == plain {
		t.Fatalf("encrypted value must differ from plain text")
	}

	decrypted, err := cipher.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("expected decrypt success, got error: %v", err)
	}
	if decrypted != plain {
		t.Fatalf("decrypted mismatch, want=%q got=%q", plain, decrypted)
	}
}
