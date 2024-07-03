package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"os"
	"os/exec"
)

type ExtensionInfo struct {
	ID        string   `json:"id"`
	Endpoints []string `json:"endpoints"`
}

type Request struct {
	Port    int               `json:"port,omitempty"`
	Path    string            `json:"path,omitempty"`
	Query   map[string]string `json:"query,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type Response struct {
	Status   int               `json:"status"`
	Headers  map[string]string `json:"headers"`
	Body     string            `json:"body"`
	Commands [][]string        `json:"commands"`
}

func main() {
	if len(os.Args) < 2 {
		return
	}

	cmd := exec.Command(os.Args[1], os.Args[2:]...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(stdin)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	decoder := json.NewDecoder(stdout)

	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	var info ExtensionInfo
	err = decoder.Decode(&info)
	if err != nil {
		panic(err)
	}

	{
		var b bytes.Buffer
		b.WriteByte(byte(len(info.ID)))
		b.WriteString(info.ID)
		b.WriteByte(byte(len(info.Endpoints)))
		for _, endpoint := range info.Endpoints {
			b.WriteByte(byte(len(endpoint)))
			b.WriteString(endpoint)
		}

		_, err = b.WriteTo(os.Stdout)
		if err != nil {
			panic(err)
		}
	}

	var data []byte
	var n int

	for {
		data = make([]byte, 3)

		n, err = io.ReadFull(os.Stdin, data)
		if err != nil {
			panic(err)
		}
		if n != 3 || data[0] != 0 {
			break
		}

		port := int(binary.NativeEndian.Uint16(data[1:]))

		var path string

		if len(info.Endpoints) > 1 {
			data = make([]byte, 1)

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != 1 {
				break
			}

			data = make([]byte, int(data[0]))

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != len(data) {
				break
			}

			path = string(data)
		}

		data = make([]byte, 4)

		n, err = io.ReadFull(os.Stdin, data)
		if err != nil {
			panic(err)
		}
		if n != 4 {
			break
		}

		queryCount := int(binary.NativeEndian.Uint32(data))
		query := map[string]string{}

		for i := 0; i < queryCount; i++ {
			data = make([]byte, 4)

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != 4 {
				return
			}

			data = make([]byte, int(binary.NativeEndian.Uint32(data)))

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != len(data) {
				return
			}

			name := string(data)

			data = make([]byte, 4)

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != 4 {
				return
			}

			data = make([]byte, int(binary.NativeEndian.Uint32(data)))

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != len(data) {
				return
			}

			query[name] = string(data)
		}

		data = make([]byte, 4)

		n, err = io.ReadFull(os.Stdin, data)
		if err != nil {
			panic(err)
		}
		if n != 4 {
			break
		}

		headerCount := int(binary.NativeEndian.Uint32(data))
		headers := map[string]string{}

		for i := 0; i < headerCount; i++ {
			data = make([]byte, 4)

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != 4 {
				return
			}

			data = make([]byte, int(binary.NativeEndian.Uint32(data)))

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != len(data) {
				return
			}

			name := string(data)

			data = make([]byte, 4)

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != 4 {
				return
			}

			data = make([]byte, int(binary.NativeEndian.Uint32(data)))

			n, err = io.ReadFull(os.Stdin, data)
			if err != nil {
				panic(err)
			}
			if n != len(data) {
				return
			}

			headers[name] = string(data)
		}

		err = encoder.Encode(Request{
			Port:    port,
			Path:    path,
			Query:   query,
			Headers: headers,
		})
		if err != nil {
			panic(err)
		}

		var res Response

		err = decoder.Decode(&res)
		if err != nil {
			panic(err)
		}

		var b bytes.Buffer
		binary.Write(&b, binary.NativeEndian, uint16(res.Status))
		b.WriteByte(byte(len(res.Headers)))
		for name, value := range res.Headers {
			b.WriteByte(byte(len(name)))
			b.WriteString(name)
			b.WriteByte(byte(len(value)))
			b.WriteString(value)
		}
		if res.Body == "" {
			binary.Write(&b, binary.NativeEndian, uint32(0))
		} else {
			body, err := base64.StdEncoding.DecodeString(res.Body)
			if err == nil {
				binary.Write(&b, binary.NativeEndian, uint32(len(body)))
				b.Write(body)
				binary.Write(&b, binary.NativeEndian, uint32(0))
			} else {
				binary.Write(&b, binary.NativeEndian, uint32(0))
			}
		}
		b.WriteByte(byte(len(res.Commands)))
		for _, c := range res.Commands {
			b.WriteByte(byte(len(c)))
			for _, part := range c {
				b.WriteByte(byte(len(part)))
				b.WriteString(part)
			}
		}

		_, err = b.WriteTo(os.Stdout)
		if err != nil {
			panic(err)
		}
	}
}
