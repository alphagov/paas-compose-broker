package broker

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"

	"github.com/alphagov/paas-compose-broker/compose"
	composeapi "github.com/compose/gocomposeapi"
)

var errDeploymentNotFound = errors.New("Deployment not found")

func findDeployment(c compose.Client, name string) (*composeapi.Deployment, error) {
	deployment, errs := c.GetDeploymentByName(name)
	if len(errs) > 0 {
		if strings.Contains(errs[0].Error(), "deployment not found") {
			return nil, errDeploymentNotFound
		}
		return nil, compose.SquashErrors(errs)
	}

	return deployment, nil
}

func makeOperationData(operationType, recipeID string) (string, error) {

	operationData := OperationData{
		Type:     operationType,
		RecipeID: recipeID,
	}

	data, err := json.Marshal(operationData)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func makeInstanceName(dbPrefix, instanceID string) (string, error) {
	if dbPrefix == "" {
		return "", errors.New("dbPrefix can't be empty")
	}
	if instanceID == "" {
		return "", errors.New("instanceID can't be empty")
	}
	return fmt.Sprintf("%s-%s", strings.TrimSpace(dbPrefix), instanceID), nil
}

func makeUserName(bindingID string) string {
	return fmt.Sprintf("user_%s", bindingID)
}

func makeDatabaseName(instanceID string) string {
	return fmt.Sprintf("db_%s", instanceID)
}

func makeRandomPassword(desired_bytes_of_entropy int) (string, error) {
	buf := make([]byte, desired_bytes_of_entropy)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	// Convert to base64. The resulting password will therefore be longer than
	// `desired_bytes_of_entropy` but ASCII is safer to send between systems than
	// arbitrary binary data.
	return base64.URLEncoding.EncodeToString(buf), nil
}

func MongoConnection(uri, caBase64 string) (*mgo.Session, error) {
	mongoUrl, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	password, _ := mongoUrl.User.Password()
	return mgo.DialWithInfo(&mgo.DialInfo{
		Addrs:    strings.Split(mongoUrl.Host, ","),
		Database: strings.TrimPrefix(mongoUrl.Path, "/"),
		Timeout:  10 * time.Second,
		Username: mongoUrl.User.Username(),
		Password: password,
		DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
			ca, err := base64.StdEncoding.DecodeString(caBase64)
			if err != nil {
				return nil, err
			}
			roots := x509.NewCertPool()
			roots.AppendCertsFromPEM(ca)
			return tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr.String(), &tls.Config{
				RootCAs: roots,
			})
		},
	})
}
