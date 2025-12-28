package handlers

import (
	"frogs_cafe/database"
)

type Handler struct {
	db *database.DB
}

func New(db *database.DB) *Handler {
	h := &Handler{db: db}
	InitHub(h)
	return h
}
