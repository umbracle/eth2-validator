package proto

/*
import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ssz "github.com/ferranbt/fastssz"
	"gopkg.in/yaml.v2"
)

type codec interface {
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
}

type testCallback func() codec

var codecs = map[string]testCallback{
	"AttestationData":    func() codec { return new(AttestationData) },
	"Checkpoint":         func() codec { return new(Checkpoint) },
	"IndexedAttestation": func() codec { return new(IndexedAttestation) },
	"PendingAttestation": func() codec { return new(PendingAttestation) },
	"DepositData":        func() codec { return new(DepositData) },
	"DepositMessage":     func() codec { return new(DepositMessage) },
}

func TestSpecMinimal(t *testing.T) {
	files := readDir(t, filepath.Join(testsPath, "/minimal/phase0/ssz_static"))
	for _, f := range files {
		spl := strings.Split(f, "/")
		name := spl[len(spl)-1]

		base, ok := codecs[name]
		if !ok {
			continue
		}

		t.Log(f)
		for _, f := range walkPath(t, f) {
			checkSSZEncoding(t, f, base)
		}
	}
}

func TestSpecMainnet(t *testing.T) {
	files := readDir(t, filepath.Join(testsPath, "/mainnet/phase0/ssz_static"))
	for _, f := range files {
		spl := strings.Split(f, "/")
		name := spl[len(spl)-1]

		if name == "BeaconState" || name == "HistoricalBatch" {
			continue
		}
		base, ok := codecs[name]
		if !ok {
			continue
		}

		t.Log(f)
		files := readDir(t, filepath.Join(f, "ssz_random"))
		for _, f := range files {
			checkSSZEncoding(t, f, base)
		}
	}
}

func checkSSZEncoding(t *testing.T, f string, base testCallback) {
	obj := base()
	output := readValidGenericSSZ(t, f, &obj)

	// Marshal
	res, err := obj.MarshalSSZTo(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(res, output.ssz) {
		t.Fatal("bad marshalling")
	}

	// Unmarshal
	obj2 := base()
	if err := obj2.UnmarshalSSZ(res); err != nil {
		panic(err)
	}
	if !deepEqual(obj, obj2) {
		t.Fatal("bad unmarshalling")
	}

	// Root
	root, err := obj.HashTreeRoot()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(root[:], output.root) {
		fmt.Printf("%s bad root\n", f)
	}
}

const (
	testsPath      = "../../eth2.0-spec-tests/tests"
	serializedFile = "serialized.ssz"
	valueFile      = "value.yaml"
	rootsFile      = "roots.yaml"
)

func walkPath(t *testing.T, path string) (res []string) {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.Contains(path, "case_") {
			res = append(res, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func readDir(t *testing.T, path string) []string {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		t.Fatal(err)
	}
	res := []string{}
	for _, f := range files {
		res = append(res, filepath.Join(path, f.Name()))
	}
	return res
}

type output struct {
	root []byte
	ssz  []byte
}

func readValidGenericSSZ(t *testing.T, path string, obj interface{}) *output {
	serialized, err := ioutil.ReadFile(filepath.Join(path, serializedFile))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := ioutil.ReadFile(filepath.Join(path, valueFile))
	if err != nil {
		t.Fatal(err)
	}
	raw2, err := ioutil.ReadFile(filepath.Join(path, rootsFile))
	if err != nil {
		t.Fatal(err)
	}

	// Decode ssz root
	var out map[string]string
	if err := yaml.Unmarshal(raw2, &out); err != nil {
		t.Fatal(err)
	}
	root, err := hex.DecodeString(out["root"][2:])
	if err != nil {
		t.Fatal(err)
	}

	if err := ssz.UnmarshalSSZTest(raw, obj); err != nil {
		t.Fatal(err)
	}
	return &output{root: root, ssz: serialized}
}
*/
