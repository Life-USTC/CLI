package me

import "testing"

func TestProfileFieldsPreserveEmailWithoutNullableAdminStatus(t *testing.T) {
	keys := make(map[string]bool)
	for _, field := range profileFields() {
		keys[field.Key] = true
	}
	if !keys["email"] {
		t.Fatal("human profile output must include email")
	}
	if keys["isAdmin"] {
		t.Fatal("human profile output must not present nullable OAuth isAdmin")
	}
}
