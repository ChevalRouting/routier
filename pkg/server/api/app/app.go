package app

import (
	"github.com/ChevalRouting/routier/pkg/auth/identity"
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/server/tlscert"
	"github.com/ChevalRouting/routier/pkg/switches"
	"github.com/go-playground/validator/v10"
)

type App struct {
	DB         *webdb.DB
	ConfigPath string
	JWTSecret  []byte
	Identity   *identity.Identity
	Switches   *switches.Engine
	Validator  *validator.Validate
	Debug      bool

	Certificates *tlscert.Provider

	AdvertisePort int
	AdvertiseTLS  bool
}
