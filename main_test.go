package main

import (
	"fmt"
	"testing"
)

func TestTagsWrong(t *testing.T) {
	testData := []string{"", " ", "BAD", "BAD;BAD2"}

	for _, v := range testData {
		t.Run(fmt.Sprintf("TAGS are set to %q", v), func(t *testing.T) {
			tags := tags(v)

			if len(tags) != 0 {
				t.Fatal()
			}
		})
	}
}

func TestTagsCorrect(t *testing.T) {
	testData := []string{"K1=V1", "K1=V1;K2=V2"}

	for i := 0; i < len(testData); i++ {
		t.Run(fmt.Sprintf("TAGS are set to %q", testData[i]), func(t *testing.T) {
			tags := tags(testData[i])

			if len(tags) != i+1 {
				t.Fatal(tags)
			}

		})
	}
}
