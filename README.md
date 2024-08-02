# HFS

HTTP Framework implemented using `GO` without any library or http/net package. this is hobby project, it's not for production. you can expect many bug.

## How to use

you can open `cmd` folder to see full example.
```go
func main() {
    server := hfs.NewServer("localhost:3000", hfs.Option{})

	server.Handle("/", func(req hfs.Request) *hfs.Response {
		return &hfs.Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/html",
			},
			Body: "Hello, World",
		}
	})

    // you can also using `hfs.NewResponse` to create response
	server.Handle("/args", func(req hfs.Request) *hfs.Response {
		response := hfs.NewResponse()
		response.SetCode(200)
        
        // get query args
        name := req.Args["name"]
		response.SetBody(name)

		return response
	})
}
```

this also support websocket.

```go
var websocket = hfs.NewWebsocket(nil)
func main() {
    ...

    server.Handle("/ws", func(req hfs.Request) *hfs.Response {
		client, err := websocket.Upgrade(req)
		if err != nil {
			panic(err)
		}

		for {
			p, err := client.Read()
			if err != nil {
                log.Println(err)
				client.Close()

			}

			err = client.Send("Hello, Client")
			if err != nil {
                log.Println(err)
				break
			}

			fmt.Println("Received: ", string(p))
		}

		return &hfs.Response{
			Code: 200,
			Headers: map[string]string{
				"Content-Type": "text/plain",
			},
			Body: "Websocket",
		}
    })
}
```

## Reference

This project based on [rfc6455](https://datatracker.ietf.org/doc/html/rfc6455) and [rfc2616](https://datatracker.ietf.org/doc/html/rfc2616). But, currently doesn't implement all features.