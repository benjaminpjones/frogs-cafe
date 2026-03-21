package handlers

import (
	"frogs_cafe/bik"
	"frogs_cafe/config"
	"frogs_cafe/database"
)

type Handler struct {
	db       *database.DB
	cfg      *config.Config
	bikKeys  *bik.Keys
	keyCache *bik.KeyCache
}

func New(db *database.DB, cfg *config.Config) (*Handler, error) {
	keys, err := bik.LoadOrGenerateKeys(cfg.BIKPrivateKey)
	if err != nil {
		return nil, err
	}
	h := &Handler{
		db:       db,
		cfg:      cfg,
		bikKeys:  keys,
		keyCache: bik.NewKeyCache(),
	}
	InitHub(h)
	return h, nil
}
