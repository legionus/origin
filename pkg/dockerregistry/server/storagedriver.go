package server

import (
	storagedriver "github.com/docker/distribution/registry/storage/driver"
	registrystorage "github.com/docker/distribution/registry/storage/driver/middleware"
)

// StorageDriver gives access to the blob store.
// This variable holds the object created by the docker/distribution. We import
// it into our namespace because there are no other ways to access it. In other
// cases it is hidden from us.
var StorageDriver storagedriver.StorageDriver

func init() {
	registrystorage.Register("openshift", func(driver storagedriver.StorageDriver, options map[string]interface{}) (storagedriver.StorageDriver, error) {
		// We can do this because of an initialization sequence of middlewary.
		// Storage driver required to create registry. So we can be sure that
		// this assignment will happen before registry and repository initialization.
		StorageDriver = driver
		return StorageDriver, nil
	})
}
