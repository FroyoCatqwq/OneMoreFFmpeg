package platform

// EnvStore is the persistent user environment backend.
type EnvStore interface {
	Get(name string) (string, bool, error)
	Set(name, value string) error
	Remove(name string) error
}
