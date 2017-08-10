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
	session *mgo.Session
}

func NewMongoEngine() *MongoEngine {
	return &MongoEngine{}
}

func (m *MongoEngine) CreateUser(instanceID, bindingID string, deployment *composeapi.Deployment) (*Credentials, error) {
	var err error

	credentials := &Credentials{}
	credentials.Name = makeDatabaseName(instanceID)
	credentials.Username = makeUserName(bindingID)
	credentials.Password, err = makeRandomPassword(passwordLength)
	if err != nil {
		return credentials, err
	}

	if m.session == nil {
		return nil, fmt.Errorf("session: not open")
	}

	err = m.session.DB(credentials.Name).UpsertUser(&mgo.User{
		Username: credentials.Username,
		Password: credentials.Password,
		Roles:    []mgo.Role{mgo.RoleReadWrite},
	})
	if err != nil {
		return credentials, err
	}

	rootCredentials, err := m.ParseConnectionString(deployment)
	if err != nil {
		return credentials, err
	}

	credentials.Host = rootCredentials.Host
	credentials.Port = rootCredentials.Port
	mongoURI := url.URL{
		Scheme: "mongodb",
		User:   url.UserPassword(credentials.Username, credentials.Password),
		Host:   credentials.Host + ":" + credentials.Port,
		Path:   credentials.Name,
	}
	credentials.URI = mongoURI.String()
	credentials.CACertificateBase64 = deployment.CACertificateBase64

	return credentials, nil
}

func (m *MongoEngine) Open(credentials *Credentials) error {
	roots := x509.NewCertPool()
	ca, err := base64.StdEncoding.DecodeString(credentials.CACertificateBase64)
	if err != nil {
		return err
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

	m.session, err = mgo.DialWithInfo(&dialInfo)

	return err
}

func (m *MongoEngine) Close() {
	m.session.Close()
}

func (m *MongoEngine) DropUser(instanceID, bindingID string, deployment *composeapi.Deployment) error {
	return m.session.DB(makeDatabaseName(instanceID)).RemoveUser(makeUserName(bindingID))
}

func (m *MongoEngine) ParseConnectionString(deployment *composeapi.Deployment) (*Credentials, error) {
	if deployment == nil {
		return nil, fmt.Errorf("no deployment provided: cannot parse the connection string")
	}

	mongoURL, err := url.Parse(deployment.Connection.Direct[0])
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
		CACertificateBase64: deployment.CACertificateBase64,
	}, nil
}
