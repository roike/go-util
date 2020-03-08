package helper

import (
	"testing"
)

func TestRandomString(t *testing.T) {
	randAlphabets := RandomString(6)
	randAlphabets2 := RandomString(6)
	if randAlphabets == randAlphabets2 {
		t.Fatal("testRandomstring failed")
	}
	t.Logf("First is %v, Second is %v", randAlphabets, randAlphabets2)
}
