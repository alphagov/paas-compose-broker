package dbengine

import composeapi "github.com/compose/gocomposeapi"

type Provider interface {
	GetDBEngine(*composeapi.Deployment) (DBEngine, error)
}
