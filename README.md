# A Go-based Dummy Authentication Plugin for Kong

### How to Execute

1. Install [Docker](Version: 20.10.12)

2. Install [Go] (Version go1.17.6)
   
3. If you make modifications to the .go file, you should run a build. Make sure you cross compile to Linux (where applicable).
	```
	$Env:GOOS = "linux"; $Env:GOARCH = "amd64"
	go env
	go build

	```
   
4. Start Docker Compose
	```
	docker-compose up -d
	
	```


### How to Test

Install one of the following: Newman (Version 5.3.1), cURL (Version 7.55.1), or Postman (version 9.10.1)


1. Run `curl -v http://host.docker.internal:8000/api/uuid -H "access_key: 14503afb-202c-47e6-b82b-f15e5f4f5f04" -H "Content-Type: application/json"'`, it will return `200` with valid data.
   
![alt tag](/status-code-200.png)


2. Run `curl -v http://host.docker.internal:8000/api/uuid -H "access_key: invalid" -H "Content-Type: application/json"'`, it will return `401`.
   
![alt tag](/status-code-401.png)

3. Check kong-test-cURL.txt for execution trace.
   
4. Verify the id_token header in JWT.io
   
![alt tag](/dummy-jwt.png)

5. Alternatively run the Postman Collection in Postman or in Newman using
   ```
	newman run gw-test.postman_collection.json -e kong-env.postman_environment.json
	
   ```

6. Check gw-test-results.txt for Newman execution trace.



### Clean-Up

1. Stop Docker Compose
   
   ```
	docker compose down
	
	```

### Known Issues

When the container is started, everything works perfectly fine. Kong, on the other hand, intermittently returns the `"no plugin instance"` error after a few runs. 

![alt tag](/error-trace.png)

Please note that we are using the Kong image: kong:2.3-alpine with Kong go-pdk v0.6.1. This was a deliberate choice as the latest combination - Kong image: kong:latest & Kong go-pdk v0.7.1 - fails with `"go-pluginserver stopped"` error. I believe the newer versions use protobuf, which has resulted in some incompatibility issues. 

So far, the only solution is to restart Kong. There appears to be an existing Kong PR for this issue (versions 2.3 and later, reference link - https://github.com/Kong/kong/issues/7148 ). However, the longer-term solution will be to rewrite the Plugin in Lua, where I do not believe the bug exists. 