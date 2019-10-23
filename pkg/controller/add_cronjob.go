package controller

import (
	"example/cronjob-operator/pkg/controller/cronjob"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, cronjob.Add)
}
