package login

import (
	"encoding/xml"
	"errors"
	"log"
	"strings"

	"github.com/jparedesimx/sii/config"
	"github.com/jparedesimx/sii/dsig"
	"github.com/jparedesimx/sii/soap"

	"github.com/antchfx/xmlquery"
)

type seed struct {
	XMLName xml.Name
	Body    struct {
		XMLName         xml.Name
		GetSeedResponse struct {
			XMLName       xml.Name
			GetSeedReturn string `xml:"getSeedReturn"`
		} `xml:"getSeedResponse"`
	}
}
type token struct {
	XMLName xml.Name
	Body    struct {
		XMLName          xml.Name
		GetTokenResponse struct {
			XMLName        xml.Name
			GetTokenReturn string `xml:"getTokenReturn"`
		} `xml:"getTokenResponse"`
	}
}

var response []byte
var err error

// AuthWebService implements SII authentication using soap webservices
func AuthWebService(certBase64 string, password string, env string) (string, error) {
	body := []byte(strings.TrimSpace(config.SeedTemplate))
	retries := 10
	for retries > 0 {
		seedURL := config.CertSeedWdsl
		if env == "production" {
			seedURL = config.ProdSeedWdsl
		}
		response, err = soap.Request(seedURL, body)
		if strings.Contains(string(response), "503") {
			err = errors.New("503 Service Unavailable")
		}
		if err != nil {
			retries--
			if retries == 0 {
				return "", err
			}
		} else {
			break
		}
	}
	// log.Println("Seed Tries: ", retries)
	// Parse response to xml struct
	var seed seed
	err = xml.Unmarshal([]byte(string(response)), &seed)
	if err != nil {
		log.Println("Error unmarshalling xml. ", err.Error())
		return "", err
	}
	responseNode, err := xmlquery.Parse(strings.NewReader(seed.Body.GetSeedResponse.GetSeedReturn))
	if err != nil {
		log.Println("Error reading seed. ", err.Error())
		return "", err
	}
	seedNode := xmlquery.FindOne(responseNode, "//SEMILLA")
	// log.Println("SEMILLA:", seedNode.InnerText())
	pszXML := strings.Replace(config.PszXML, "@seed", seedNode.InnerText(), 1)
	// Sign pszXML and return the generated file like a byte array
	pszSigned, err := dsig.Sign(certBase64, password, pszXML)
	if err != nil {
		return "", err
	}
	pszXML = strings.Replace(config.TokenTemplate, "@pszXML", string(pszSigned), 1)
	body = []byte(strings.TrimSpace(pszXML))
	retries = 10
	for retries > 0 {
		tokenURL := config.CertTokenWsdl
		if env == "production" {
			tokenURL = config.ProdTokenWsdl
		}
		response, err = soap.Request(tokenURL, body)
		if strings.Contains(string(response), "503") {
			err = errors.New("503 Service Unavailable")
		}
		if err != nil {
			retries--
			if retries == 0 {
				return "", err
			}
		} else {
			break
		}
	}
	// log.Println("Token Tries: ", retries)
	// Parse response to xml struct
	var token token
	err = xml.Unmarshal([]byte(string(response)), &token)
	if err != nil {
		log.Println("Error unmarshalling xml. ", err.Error())
		return "", err
	}
	responseNode, err = xmlquery.Parse(strings.NewReader(token.Body.GetTokenResponse.GetTokenReturn))
	if err != nil {
		log.Println("Error reading token. ", err.Error())
		return "", err
	}
	tokenNode := xmlquery.FindOne(responseNode, "//TOKEN")
	// log.Println("TOKEN:", tokenNode.InnerText())
	return tokenNode.InnerText(), nil
}
