package dbengine

type Provider interface {
	GetDBEngine(engine string) (DBEngine, error)
}
