package config

const (
	defaultDatastoreType = "levelds"
	defaultDatastoreDir  = "datastore"
)

// Datastore tracks the configuration of the datastore.
type Datastore struct {
	// Type is the type of datastore.
	Type string
	// Dir is the directory where the datastore is kept. If this is not an
	// absolute path then the location is relative to the indexer repo
	// directory.
	Dir string
}
