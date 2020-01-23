package main

import (
	"flag"
	"fmt"
	"os"
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

func TestEnvOverFlags(t *testing.T) {
	err := os.Setenv("URL", "http://localhost")
	if err != nil {
		t.Fatal(err)
	}

	var testFlag = flag.String("url", DEFAULT_ANODOT_URL, "url value")
	err = flag.Set("url", "http://192.168.0.1")
	if err != nil {
		t.Fatal(err)
	}

	v := envOrFlag("URL", testFlag)
	if v != "http://localhost" {
		t.Fatal("env variable should have higher order over flags")
	}
}

func TestFlagEnvNotSet(t *testing.T) {
	var testFlag = flag.String("url", DEFAULT_ANODOT_URL, "url value")
	err := flag.Set("url", "http://192.168.0.1")
	if err != nil {
		t.Fatal(err)
	}

	v := envOrFlag("URL", testFlag)
	if v != "http://192.168.0.1" {
		t.Fatal("flag value should be used is env variable is not set")
	}
}
