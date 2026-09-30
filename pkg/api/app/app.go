package app

import (
	webdb "github.com/ChevalRouting/routier/pkg/db"
	"github.com/ChevalRouting/routier/pkg/identity"
	"github.com/go-playground/validator/v10"
)

type App struct {
	DB         *webdb.DB
	ConfigPath string
	JWTSecret  []byte
	Identity   *identity.Identity
	Validator  *validator.Validate
	Debug      bool

	AdvertisePort int
	AdvertiseTLS  bool
}
