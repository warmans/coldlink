package coldlink

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestTempFile(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "test-tempfile")
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	tf := TempFile{f}
	name := tf.Name()
	err = tf.Close()
	if err != nil {
		t.Errorf("expected nil, got %s", err)
	}
	_, err = os.Open(name)
	if err == nil {
		t.Errorf("expected err, got nil")
	}
}
