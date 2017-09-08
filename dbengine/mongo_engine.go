package dbengine

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	composeapi "github.com/compose/gocomposeapi"
	mgo "gopkg.in/mgo.v2"
)

const (
	passwordLength = 32
)

type MongoEngine struct {
	deployment *composeapi.Deployment
}

func NewMongoEngine(deployment *composeapi.Deployment) *MongoEngine {
	return &MongoEngine{deployment}
}

func (e *MongoEngine) GenerateCredentials(instanceID, bindingID string) (*Credentials, error) {

	masterCredentials, err := e.getMasterCredentials()
	if err != nil {
		return nil, err
	}
	session, err := newMongoSession(masterCredentials)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	dbname := makeDatabaseName(instanceID)
	username := makeUserName(bindingID)
	password, err := makeRandomPassword(passwordLength)
	if err != nil {
		return nil, err
	}

	err = session.DB(dbname).UpsertUser(&mgo.User{
		Username: username,
		Password: password,
		Roles:    []mgo.Role{mgo.RoleReadWrite},
	})
	if err != nil {
		return nil, err
	}

	return &Credentials{
		Host:     masterCredentials.Host,
		Port:     masterCredentials.Port,
		Name:     dbname,
		Username: username,
		Password: password,
		URI: (&url.URL{
			Scheme: "mongodb",
			User:   url.UserPassword(username, password),
			Host:   masterCredentials.Host + ":" + masterCredentials.Port,
			Path:   dbname,
		}).String(),
		CACertificateBase64: e.deployment.CACertificateBase64,
	}, nil
}

func newMongoSession(credentials *Credentials) (*mgo.Session, error) {
	roots := x509.NewCertPool()
	ca, err := base64.StdEncoding.DecodeString(credentials.CACertificateBase64)
	if err != nil {
		return nil, err
	}

	roots.AppendCertsFromPEM(ca)
	tlsConfig := &tls.Config{}
	tlsConfig.RootCAs = roots

	dialInfo := mgo.DialInfo{
		Addrs:    []string{credentials.Host + ":" + credentials.Port},
		Database: credentials.Name,
		Timeout:  10 * time.Second,
		Username: credentials.Username,
		Password: credentials.Password,
		DialServer: func(addr *mgo.ServerAddr) (net.Conn, error) {
			conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
			return conn, err
		},
	}

	return mgo.DialWithInfo(&dialInfo)
}

func (e *MongoEngine) RevokeCredentials(instanceID, bindingID string) error {
	masterCredentials, err := e.getMasterCredentials()
	if err != nil {
		return err
	}
	session, err := newMongoSession(masterCredentials)
	if err != nil {
		return err
	}
	defer session.Close()
	return session.DB(makeDatabaseName(instanceID)).RemoveUser(makeUserName(bindingID))
}

func (e *MongoEngine) getMasterCredentials() (*Credentials, error) {
	if e.deployment == nil {
		return nil, fmt.Errorf("no deployment provided: cannot parse the connection string")
	} else if len(e.deployment.Connection.Direct) < 1 {
		return nil, fmt.Errorf("failed to get connection string")
	}

	mongoURL, err := url.Parse(e.deployment.Connection.Direct[0])
	if err != nil {
		return nil, err
	}
	password, _ := mongoURL.User.Password()
	return &Credentials{
		Host: mongoURL.Hostname(),
		// FIXME: Follow up story should fix mongo connection string handling.
		// Right now we are hardcoding first host from the comma delimited list that Compose provides.
		// url.Parse() parses mongo connection string wrongly and doesn't return an error
		// so url.Port() returns port like "18899,aws-eu-west-1-portal.7.dblayer.com:18899"
		Port:                strings.Split(mongoURL.Port(), ",")[0],
		URI:                 mongoURL.String(),
		Username:            mongoURL.User.Username(),
		Password:            password,
		Name:                strings.Split(mongoURL.EscapedPath(), "/")[1],
		CACertificateBase64: e.deployment.CACertificateBase64,
	}, nil
}
