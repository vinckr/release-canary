package driver

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/ory/hydra/driver/config"
	"github.com/ory/x/configx"
	"github.com/ory/x/logrusx"

	"github.com/gorilla/sessions"
)

func TestRegistryBase_newKeyStrategy_handlesNetworkError(t *testing.T) {
	// Test ensures any network specific error is logged with a
	// specific message when attempting to create a new key strategy: issue #2338

	hook := test.Hook{} // Test hook for asserting log messages

	l := logrusx.New("", "", logrusx.WithHook(&hook))
	l.Logrus().SetOutput(ioutil.Discard)
	l.Logrus().ExitFunc = func(int) {} // Override the exit func to avoid call to os.Exit

	// Create a config and set a valid but unresolvable DSN
	c := config.MustNew(l, configx.WithConfigFiles("../internal/.hydra.yaml"))
	c.MustSet(config.KeyDSN, "postgres://user:password@127.0.0.1:9999/postgres")
	c.MustSet(config.HsmEnabled, "false")

	registry, err := NewRegistryFromDSN(context.Background(), c, l)
	if err != nil {
		t.Error("failed to create registry: ", err)
		return
	}

	registryBase := RegistryBase{r: registry, l: l}
	registryBase.WithConfig(c)

	strategy := registryBase.newKeyStrategy("key")

	assert.Equal(t, nil, strategy)
	assert.Equal(t, logrus.FatalLevel, hook.LastEntry().Level)
	assert.Contains(t, hook.LastEntry().Message, "A network error occurred, see error for specific details.")
}

func TestRegistryBase_CookieStore_MaxAgeZero(t *testing.T) {
	// Test ensures that CookieStore MaxAge option is equal to zero after initialization

	r := new(RegistryBase)
	r.WithConfig(config.MustNew(logrusx.New("", "")))

	cs := r.CookieStore().(*sessions.CookieStore)

	assert.Equal(t, cs.Options.MaxAge, 0)
}
