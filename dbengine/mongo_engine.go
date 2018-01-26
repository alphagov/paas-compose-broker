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

func dialInfoToCredentials(dialInfo *mgo.DialInfo, caCertificateBase64 string) *Credentials {
	baseURI := url.URL{
		Scheme: "mongodb",
		User:   url.UserPassword(dialInfo.Username, dialInfo.Password),
		Host:   "mongo-db-host-place-holder",
		Path:   dialInfo.Database,
	}

	mongoURI := strings.Replace(
		baseURI.String(),
		"mongo-db-host-place-holder",
		strings.Join(dialInfo.Addrs, ","),
		-1,
	)

	return &Credentials{
		Hosts:               dialInfo.Addrs,
		Name:                dialInfo.Database,
		Username:            dialInfo.Username,
		Password:            dialInfo.Password,
		URI:                 mongoURI,
		CACertificateBase64: caCertificateBase64,
	}
}

func (e *MongoEngine) GenerateCredentials(instanceID, bindingID string) (*Credentials, error) {
	masterDialInfo, err := e.getMasterDialInfo()
	if err != nil {
		return nil, err
	}

	session, err := mgo.DialWithInfo(masterDialInfo)
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

	bindDialInfo := masterDialInfo
	bindDialInfo.Database = dbname
	bindDialInfo.Username = username
	bindDialInfo.Password = password

	return dialInfoToCredentials(bindDialInfo, e.deployment.CACertificateBase64), nil
}

func (e *MongoEngine) RevokeCredentials(instanceID, bindingID string) error {
	masterDialInfo, err := e.getMasterDialInfo()
	if err != nil {
		return err
	}
	session, err := mgo.DialWithInfo(masterDialInfo)
	if err != nil {
		return err
	}
	defer session.Close()
	return session.DB(makeDatabaseName(instanceID)).RemoveUser(makeUserName(bindingID))
}

func (e *MongoEngine) getMasterDialInfo() (*mgo.DialInfo, error) {
	if e.deployment == nil {
		return nil, fmt.Errorf("no deployment provided: cannot parse the connection string")
	} else if len(e.deployment.Connection.Direct) < 1 {
		return nil, fmt.Errorf("failed to get connection string")
	} else if e.deployment.Connection.Direct[0] == "" {
		return nil, fmt.Errorf("connection string is empty")
	}

	u, err := removeSSLOption(e.deployment.Connection.Direct[0])
	if err != nil {
		return nil, err
	}

	dialInfo, err := mgo.ParseURL(u)
	if err != nil {
		return nil, err
	}
	dialInfo.Timeout = 10 * time.Second
	dialInfo.DialServer, err = createDialServer(e.deployment.CACertificateBase64)
	if err != nil {
		return nil, err
	}

	return dialInfo, nil
}

func removeSSLOption(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return uri, err
	}
	q, _ := url.ParseQuery(u.RawQuery)
	q.Del("ssl")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func createDialServer(caCert string) (func(*mgo.ServerAddr) (net.Conn, error), error) {
	roots := x509.NewCertPool()
	ca, err := base64.StdEncoding.DecodeString(caCert)
	if err != nil {
		return nil, err
	}
	roots.AppendCertsFromPEM(ca)

	tlsConfig := &tls.Config{}
	tlsConfig.RootCAs = roots

	return func(addr *mgo.ServerAddr) (net.Conn, error) {
		conn, err := tls.Dial("tcp", addr.String(), tlsConfig)
		return conn, err
	}, nil
}
