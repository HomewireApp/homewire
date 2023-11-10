package homewire_test

import (
	"fmt"
	"path"
	"testing"

	"github.com/HomewireApp/homewire"
	"github.com/HomewireApp/homewire/options"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func openWithRandomDb(t *testing.T) *homewire.Homewire {
	tmpDir := t.TempDir()
	tmpFile := path.Join(tmpDir, "test.db")

	opts, err := options.Default()
	assert.Nil(t, err)

	result, err := homewire.InitWithOptions(*opts.WithDatabasePath(tmpFile))
	assert.Nil(t, err)

	return result
}

func TestOpen(t *testing.T) {
	hw := openWithRandomDb(t)
	defer hw.Destroy()

	wires := hw.ListWires()
	assert.Equal(t, 0, len(wires), "Unexpected number of wires found")
}

func TestCreate(t *testing.T) {
	hw := openWithRandomDb(t)
	defer hw.Destroy()

	wireName := fmt.Sprintf("test-%s", uuid.NewString())
	wire, err := hw.CreateNewWire(wireName)

	assert.Equal(t, wire.Name, wireName, "Wire not created with the given name")

	if err != nil {
		t.Error(err)
	}

	wires := hw.ListWires()
	if wireCount := len(wires); wireCount != 1 {
		t.Errorf("Unexpected number of wires: %v", wireCount)
	}

	assert.Equal(t, 1, len(wires), "Unexpected number of wires found")

	firstWire := wires[0]
	assert.NotNil(t, firstWire, "First wire must not be nil")

	assert.Equal(t, wire.Name, firstWire.Name, "Wire names do not match")
	assert.Equal(t, wire.Id, firstWire.Id, "Wire IDs do not match")
}
