package util

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnmarshalFile(t *testing.T) {
	type Test struct {
		A string
	}
	test := Test{A: "hello"}
	data, err := json.MarshalIndent(test, "", " ")
	require.NoError(t, err)

	tmpfile, err := ioutil.TempFile("/tmp", "")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	_, err = tmpfile.Write(data)
	require.NoError(t, err)

	result := &Test{}
	err = UnmarshalFile(tmpfile.Name(), result)
	require.NoError(t, err)
}

// Test unmarshalling an empty file
func TestUnmarshalFileEmpty(t *testing.T) {
	type Test struct {
		A string
	}
	test := &Test{}
	err := UnmarshalFile("foo.json", test)
	require.Error(t, err)
}
