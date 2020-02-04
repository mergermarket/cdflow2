package util_test

import (
	"log"
	"reflect"
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
