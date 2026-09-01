package ldap

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-ldap/ldap/v3"
)

// Client wraps an LDAP connection and its search configuration.
type Client struct {
	Attributes         []string
	Base               string
	BindDN             string
	BindPassword       string
	GroupFilter        string
	Host               string
	ServerName         string
	UserFilter         string
	Conn               *ldap.Conn
	Port               int
	InsecureSkipVerify bool
	UseSSL             bool
	SkipTLS            bool
	ClientCertificates []tls.Certificate
}

// Connect establishes a connection to the LDAP server.
func (lc *Client) Connect() error {
	var l *ldap.Conn
	var err error
	address := fmt.Sprintf("%s:%d", lc.Host, lc.Port)

	if !lc.UseSSL {
		l, err = ldap.Dial("tcp", address)
		if err != nil {
			return err
		}

		if !lc.SkipTLS {
			err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
			if err != nil {
				return err
			}
		}
	} else {
		config := &tls.Config{
			InsecureSkipVerify: lc.InsecureSkipVerify,
			ServerName:         lc.ServerName,
		}
		if len(lc.ClientCertificates) > 0 {
			config.Certificates = lc.ClientCertificates
		}
		l, err = ldap.DialTLS("tcp", address, config)
		if err != nil {
			return err
		}
	}

	lc.Conn = l
	return nil
}

// Close closes the LDAP connection.
func (lc *Client) Close() {
	if lc.Conn != nil {
		lc.Conn.Close()
		lc.Conn = nil
	}
}

// Authenticate verifies the username and password against LDAP.
// On success it returns the user's attributes.
func (lc *Client) Authenticate(username, password string) (bool, map[string]string, error) {
	if err := lc.Connect(); err != nil {
		return false, nil, err
	}
	defer lc.Close()

	if lc.BindDN != "" && lc.BindPassword != "" {
		if err := lc.Conn.Bind(lc.BindDN, lc.BindPassword); err != nil {
			return false, nil, err
		}
	}

	attributes := append(lc.Attributes, "dn")
	searchRequest := ldap.NewSearchRequest(
		lc.Base,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf(lc.UserFilter, username),
		attributes,
		nil,
	)

	sr, err := lc.Conn.Search(searchRequest)
	if err != nil {
		return false, nil, err
	}

	if len(sr.Entries) < 1 {
		return false, nil, errors.New("user does not exist")
	}
	if len(sr.Entries) > 1 {
		return false, nil, errors.New("too many entries returned")
	}

	userDN := sr.Entries[0].DN
	user := make(map[string]string)
	for _, attr := range lc.Attributes {
		user[attr] = sr.Entries[0].GetAttributeValue(attr)
	}

	if err := lc.Conn.Bind(userDN, password); err != nil {
		return false, user, err
	}

	return true, user, nil
}

// GroupsOfUser returns the LDAP groups the user is a member of.
func (lc *Client) GroupsOfUser(username string) ([]string, error) {
	if err := lc.Connect(); err != nil {
		return nil, err
	}
	defer lc.Close()

	searchRequest := ldap.NewSearchRequest(
		lc.Base,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0, 0, false,
		fmt.Sprintf(lc.GroupFilter, username),
		[]string{"cn"},
		nil,
	)

	sr, err := lc.Conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	groups := make([]string, 0, len(sr.Entries))
	for _, entry := range sr.Entries {
		groups = append(groups, entry.GetAttributeValue("cn"))
	}
	return groups, nil
}
