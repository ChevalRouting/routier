package app

import (
	"database/sql"

	"github.com/ChevalRouting/routier/pkg/identity"
	"github.com/go-playground/validator/v10"
)

type App struct {
	DB         *sql.DB
	ConfigPath string
	JWTSecret  []byte
	Identity   *identity.Identity
	Validator  *validator.Validate
	Debug      bool

	AdvertisePort int
	AdvertiseTLS  bool
}
