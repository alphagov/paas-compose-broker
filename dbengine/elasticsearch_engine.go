package dbengine

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	composeapi "github.com/compose/gocomposeapi"
)

func isOptSep(c rune) bool {
	return c == ';' || c == '&'
}

type urlInfo struct {
	addrs []string
	user  string
	pass  string
	db    string
}

func extractURL(s string) (*urlInfo, error) {
	prefix := "http://"
	if strings.HasPrefix(s, prefix) {
		s = s[len(prefix):]
	}
	info := &urlInfo{}
	if c := strings.Index(s, "@"); c != -1 {
		pair := strings.SplitN(s[:c], ":", 2)
		if len(pair) > 2 || pair[0] == "" {
			return nil, errors.New("credentials must be provided as user:pass@host")
		}
		var err error
		info.user, err = url.QueryUnescape(pair[0])
		if err != nil {
			return nil, fmt.Errorf("cannot unescape username in URL: %q", pair[0])
		}
		if len(pair) > 1 {
			info.pass, err = url.QueryUnescape(pair[1])
			if err != nil {
				return nil, fmt.Errorf("cannot unescape password in URL")
			}
		}
		s = s[c+1:]
	}
	if c := strings.Index(s, "/"); c != -1 {
		info.db = s[c+1:]
		s = s[:c]
	}
	info.addrs = strings.Split(s, ",")

	// FIXME: this resolves an issue with the hostname returned by Compose
	// Compose may return: `cluster-name-c002.compose.direct`,
	// when it should return: `cluster-name-c00.2.compose.direct`.
	// A support ticket has been raised. This can be removed once the upstream support ticket is resolved.
	re := regexp.MustCompile(`(.+-[a-z]{1}[0-9]{2})\.?(\d+)([^:]+):(\d+)`)
	for idx, u := range info.addrs {
		if strings.Contains(u, "compose.direct") {
			faulty := re.FindStringSubmatch(u)
			info.addrs[idx] = fmt.Sprintf("%s.%s%s:%s", faulty[1], faulty[2], faulty[3], faulty[4])
		}
	}

	return info, nil
}

func parseURL(url string) (*Credentials, error) {
	uinfo, err := extractURL(url)
	if err != nil {
		return nil, err
	}

	info := Credentials{
		Hosts:    uinfo.addrs,
		Name:     uinfo.db,
		Username: uinfo.user,
		Password: uinfo.pass,
	}
	return &info, nil
}

type ElasticSearchEngine struct {
	deployment *composeapi.Deployment
}

func NewElasticSearchEngine(deployment *composeapi.Deployment) *ElasticSearchEngine {
	return &ElasticSearchEngine{deployment}
}

func (e *ElasticSearchEngine) GenerateCredentials(instanceID, bindingID string) (*Credentials, error) {
	if e.deployment == nil {
		return nil, fmt.Errorf("no deployment provided: cannot parse the connection string")
	} else if len(e.deployment.Connection.Direct) < 1 {
		return nil, fmt.Errorf("failed to get connection string")
	}

	credentials, err := parseURL(e.deployment.Connection.Direct[0])
	if err != nil {
		return nil, err
	} else if credentials.Username == "" {
		return nil, fmt.Errorf("connection string did not contain a user")
	}
	credentials.CACertificateBase64 = e.deployment.CACertificateBase64

	return credentials, nil
}

func (e *ElasticSearchEngine) RevokeCredentials(instanceID, bindingID string) error {
	return nil
}
