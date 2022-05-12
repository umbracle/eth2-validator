package structs

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
	"github.com/golang/snappy"

	"gopkg.in/yaml.v2"
)

type codec interface {
	ssz.Marshaler
	ssz.Unmarshaler
	ssz.HashRoot
}

type codecTree interface {
	GetTreeWithWrapper(w *ssz.Wrapper) (err error)
	GetTree() (*ssz.Node, error)
}

type testCallback func() codec

var codecs = map[string]testCallback{
	"AttestationData":         func() codec { return new(AttestationData) },
	"Checkpoint":              func() codec { return new(Checkpoint) },
	"AggregateAndProof":       func() codec { return new(AggregateAndProof) },
	"Attestation":             func() codec { return new(Attestation) },
	"AttesterSlashing":        func() codec { return new(AttesterSlashing) },
	"BeaconBlock":             func() codec { return new(BeaconBlock) },
	"BeaconBlockBody":         func() codec { return new(BeaconBlockBody) },
	"BeaconBlockHeader":       func() codec { return new(BeaconBlockHeader) },
	"Deposit":                 func() codec { return new(Deposit) },
	"DepositData":             func() codec { return new(DepositData) },
	"DepositMessage":          func() codec { return new(DepositMessage) },
	"Eth1Data":                func() codec { return new(Eth1Data) },
	"Fork":                    func() codec { return new(Fork) },
	"HistoricalBatch":         func() codec { return new(HistoricalBatch) },
	"IndexedAttestation":      func() codec { return new(IndexedAttestation) },
	"PendingAttestation":      func() codec { return new(PendingAttestation) },
	"ProposerSlashing":        func() codec { return new(ProposerSlashing) },
	"SignedBeaconBlock":       func() codec { return new(SignedBeaconBlock) },
	"SignedBeaconBlockHeader": func() codec { return new(SignedBeaconBlockHeader) },
	"SignedVoluntaryExit":     func() codec { return new(SignedVoluntaryExit) },
	"SigningRoot":             func() codec { return new(SigningRoot) },
	"Validator":               func() codec { return new(Validator) },
	"VoluntaryExit":           func() codec { return new(VoluntaryExit) },
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

		t.Run(name, func(t *testing.T) {
			for _, f := range walkPath(t, f) {
				checkSSZEncoding(t, f, name, base)
			}
		})
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

		t.Run(name, func(t *testing.T) {
			files := readDir(t, filepath.Join(f, "ssz_random"))
			for _, f := range files {
				checkSSZEncoding(t, f, name, base)
			}
		})
	}
}

func formatSpecFailure(errHeader, specFile, structName string, err error) string {
	return fmt.Sprintf("%s spec file=%s, struct=%s, err=%v",
		errHeader, specFile, structName, err)
}

func checkSSZEncoding(t *testing.T, fileName, structName string, base testCallback) {
	obj := base()
	output := readValidGenericSSZ(t, fileName, &obj)

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
		t.Fatal(formatSpecFailure("UnmarshalSSZ error", fileName, structName, err))
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
		fmt.Printf("%s bad root\n", fileName)
	}

	if objt, ok := obj.(codecTree); ok {
		// node root
		node, err := objt.GetTree()
		if err != nil {
			t.Fatal(err)
		}

		xx := node.Hash()
		if !bytes.Equal(xx, root[:]) {
			t.Fatal("bad node")
		}
	}
}

const (
	testsPath      = "../../../eth2.0-spec-tests/tests"
	serializedFile = "serialized.ssz_snappy"
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
	serializedSnappy, err := ioutil.ReadFile(filepath.Join(path, serializedFile))
	if err != nil {
		t.Fatal(err)
	}
	serialized, err := snappy.Decode(nil, serializedSnappy)
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
