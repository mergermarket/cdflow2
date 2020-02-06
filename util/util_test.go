package util_test

import (
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/mergermarket/cdflow2/util"
)

func TestGetEnv(t *testing.T) {
	expected := map[string]string{
		"a": "1",
		"b": "2",
	}
	if !reflect.DeepEqual(util.GetEnv([]string{"a=1", "b=2"}), expected) {
		log.Fatalln("unexpected result of GetEnv:", expected)
	}
}

func TestRandomName(test *testing.T) {
	randomName := util.RandomName("foo")
	if !strings.HasPrefix(randomName, "foo-") {
		log.Fatalln("unexpected prefix:", randomName)
	}
}
