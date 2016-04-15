package server

import (
	storagedriver "github.com/docker/distribution/registry/storage/driver"
	registrystorage "github.com/docker/distribution/registry/storage/driver/middleware"
)

func init() {
	registrystorage.Register("openshift", registrystorage.InitFunc(newLocalStorageDriver))
}

// localStorageDriver wraps storage driver of docker/distribution.
type localStorageDriver struct {
	storagedriver.StorageDriver
}

var _ storagedriver.StorageDriver = &localStorageDriver{}
var StorageDriver *localStorageDriver

func newLocalStorageDriver(storageDriver storagedriver.StorageDriver, options map[string]interface{}) (storagedriver.StorageDriver, error) {
	// We can do this because of an initialization sequence of middlewary.
	// Storage driver required to create registry. So we can be sure that
	// this assignment will happen before registry and repository initialization.
	StorageDriver = &localStorageDriver{storageDriver}
	return storageDriver, nil
}
