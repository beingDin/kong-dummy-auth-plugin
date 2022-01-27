/*
	A "dummy" Authentication Plugin in Go, which
	- Reads an opaque access key from incoming request’s HTTP Header
	- Calls the dummy verification endpoint to validate this access key
    - Based on the response from the dummy verification endpoint, updates the HTTP Headers of the API request before
	  proxying it to Backend (url: http://httpbin.org/uuid)
*/

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Kong/go-pdk"
	"github.com/Kong/go-pdk/server"
	"github.com/dgrijalva/jwt-go"
)

func main() {
	server.StartServer(New, Version, Priority)
}

var Version = "0.2"
var Priority = 1

// Declare Plugin Configurable Parameters
type Config struct {
	Key_In_Header  bool
	Key_Name       string
	Endpoint       string
	Scope          string
	Identity_Token bool
}

// Declare Custom Claims for JWT Generation
type customClaims struct {
	Content string
	Id      string
	jwt.StandardClaims
}

func New() interface{} {
	return &Config{}
}

func (conf Config) Access(kong *pdk.PDK) {

	respHeaders := make(map[string][]string)
	respHeaders["Content-Type"] = append(respHeaders["Content-Type"], "application/json")

	// Check API Access Key location. If the Key is not in the correct location, an error will be returned.
	keyExists := conf.Key_In_Header
	if keyExists == false {
		kong.Log.Info("Key Exists:", keyExists)
		kong.Response.Exit(403, "API Key location not specified", respHeaders)
	}

	// Read API Key from specified location. An error will be returened, if there's no matching API Key.
	apiKey := conf.Key_Name
	kong.Log.Info("API Key Name:", apiKey)
	accessKey, err := kong.Request.GetHeader(apiKey)

	if err != nil {
		kong.Log.Err(err.Error())
		kong.Response.Exit(403, "No matching API Key found in Header", respHeaders)
	}

	//Formulate the Target URL for Dummy Auth Server.
	authenticationURL := conf.Endpoint
	scope := conf.Scope
	targetURL := authenticationURL + accessKey + "/" + scope
	kong.Log.Info("Target URL:", targetURL)

	// Create connection request to Dummy Auth Server.
	authServerConnect := &http.Client{}
	req, err := http.NewRequest("GET", targetURL, nil)
	if err != nil {
		kong.Log.Err(err.Error())
		kong.Response.Exit(500, "Invalid Authentication URL", respHeaders)
	}

	//Invoke Target URL and verify API Access Key.
	resp, err := authServerConnect.Do(req)
	if err != nil {
		kong.Log.Err(err.Error())
		kong.Response.Exit(503, "Failed to connect to Auth Server using Authentication URL", respHeaders)
	}

	//Read Response Body returned by Auth Server.
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		kong.Log.Err(err.Error())
		kong.Response.Exit(500, "Failed to read response from Auth Server using Authentication URL", respHeaders)
	}

	// Return an error if API Key Verification fails at remote Auth Server.

	/* The following assumptions were made to simulate the Key verification process:
	   - An initial call is made to POST http://mockbin.com/bin/create. This returns a Bin ID.
	   - This Bin ID is passed in the incoming request using a configurable HTTP Header (say access_key).
	   - The Plugin reaches out to the remote server at GET http://mockbin.com/bin/{bin-id}/view endpoint.
	   - The /view (remote server) endpoint is a configurable parameter in the Plugin implementation.
	   - The access_key verification is considered successful if the /view endpoint returns valid response, and the request is allowed to be proxied.
	   - For an empty response, the client gets a “401” response.
	*/
	if len(string(bodyBytes)) == 0 {
		kong.Response.Exit(401, "Failed to Authenticate API Consumer", respHeaders)
	}

	//Set API Access Key Header.
	kong.Log.Info("API Access Key:", accessKey)
	kong.Response.SetHeader(conf.Key_Name, accessKey)

	// Based on a configurable parameter, include Identity Token Header so it is proxied with the request to the Backend.
	id_token := conf.Identity_Token
	kong.Log.Info("Include Identity Token:", id_token)

	if id_token == true {
		// Retrieve a key from the remote Server Response.
		var result map[string]interface{}
		json.Unmarshal([]byte(bodyBytes), &result)
		content := result["content"].(map[string]interface{})
		contentText := content["text"].(string)
		kong.Log.Info("Content Text returned by Auth Server:", contentText)

		/*Generate JWT with Claim Body containing
		- Content : Text retured by Remote Server.
		- API Access Key : API Key recevied in the incoming request’s HTTP Header.
		*/
		signedToken, err := GenerateJWT(contentText, accessKey)
		if err != nil {
			kong.Log.Err("Error Generating JWT")
			kong.Log.Err(err.Error())
		}

		// Include the JWT token header
		kong.Log.Info("JWT:", signedToken)
		kong.Response.SetHeader("id_token", signedToken)
	}

}

// Generate JWT with Custom Cliams and Sign it with a Dummy Secret
func GenerateJWT(content, id string) (string, error) {
	claims := customClaims{
		Content: content,
		Id:      id,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: 15000,
			Issuer:    "Kong",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte("dummy-secret"))

	if err != nil {
		return "error", err
	}
	return "Bearer " + signedToken, nil
}
