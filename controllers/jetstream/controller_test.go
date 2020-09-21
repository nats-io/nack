package jetstream

import (
	"bytes"
	"reflect"
	"testing"

	jsmapi "github.com/nats-io/jsm.go/api"
)

func TestGetStorageType(t *testing.T) {
	t.Parallel()

	cases := []struct {
		storage string

		wantType jsmapi.StorageType
		wantErr  bool
	}{
		{storage: "memory", wantType: jsmapi.MemoryStorage},
		{storage: "file", wantType: jsmapi.FileStorage},
		{storage: "junk", wantErr: true},
	}
	for _, c := range cases {
		c := c
		t.Run(c.storage, func(t *testing.T) {
			t.Parallel()

			got, err := getStorageType(c.storage)
			if err != nil && !c.wantErr {
				t.Error("unexpected error")
				t.Fatalf("got=%s; want=nil", err)
			} else if err == nil && c.wantErr {
				t.Error("unexpected success")
				t.Fatalf("got=nil; want=err")
			}

			if got != c.wantType {
				t.Error("unexpected storage type")
				t.Fatalf("got=%v; want=%v", got, c.wantType)
			}
		})
	}
}

func TestWipeSlice(t *testing.T) {
	bs := []byte("hello")
	wipeSlice(bs)
	if want := []byte("xxxxx"); !bytes.Equal(bs, want) {
		t.Error("unexpected slice wipe")
		t.Fatalf("got=%s; want=%s", bs, want)
	}
}

func TestAddFinalizer(t *testing.T) {
	fs := []string{"foo", "bar"}
	fs = addFinalizer(fs, "fizz")
	fs = addFinalizer(fs, "fizz")
	fs = addFinalizer(fs, "fizz")

	want := []string{"foo", "bar", "fizz"}
	if !reflect.DeepEqual(fs, want) {
		t.Error("unexpected finalizers")
		t.Fatalf("got=%v; want=%v", fs, want)
	}
}

func TestRemoveFinalizer(t *testing.T) {
	fs := []string{"foo", "bar"}
	fs = removeFinalizer(fs, "bar")
	fs = removeFinalizer(fs, "bar")
	fs = removeFinalizer(fs, "bar")

	want := []string{"foo"}
	if !reflect.DeepEqual(fs, want) {
		t.Error("unexpected finalizers")
		t.Fatalf("got=%v; want=%v", fs, want)
	}
}
