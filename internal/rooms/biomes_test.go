package rooms

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestBiomeInfo_IndoorFlag(t *testing.T) {
	yamlSrc := []byte("biomeid: testcave\nname: Test Cave\nsymbol: \"^\"\nindoor: true\n")
	var bi BiomeInfo
	if err := yaml.Unmarshal(yamlSrc, &bi); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !bi.Indoor {
		t.Error("expected indoor: true to set BiomeInfo.Indoor")
	}

	var bi2 BiomeInfo
	if err := yaml.Unmarshal([]byte("biomeid: plains\nname: Plains\nsymbol: \".\"\n"), &bi2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if bi2.Indoor {
		t.Error("expected Indoor to default false")
	}
}
