package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strings"
)

const (
	baseURL   = "hw1.alexbers.com"
	userToken = "e238c3c3730304c53a49f7ca0c04ce63"
	tries     = 100
)

type RequestData struct {
	Method      string
	Path        string
	Cookies     map[string]string
	Headers     map[string]string
	Form        map[string]string
	Files       map[string]string
	QueryParams map[string]string
}

func main() {
	data := RequestData{
		Method:  "GET",
		Path:    "/",
		Cookies: map[string]string{"user": userToken},
	}

	for i := 1; i <= tries; i++ {
		fmt.Printf("\n--- Step #%d ---\n", i)

		html := doRequest(data)

		fmt.Println(html)

		// if strings.Contains(html, "токен") || strings.Contains(html, "token") {
		// 	fmt.Println("Succes. Server response:")
		// 	fmt.Println(html)
		// 	break
		// }

		nextData := RequestData{
			Method:      "GET",
			Cookies:     map[string]string{"user": userToken},
			Headers:     make(map[string]string),
			Form:        make(map[string]string),
			Files:       make(map[string]string),
			QueryParams: make(map[string]string),
		}

		if strings.Contains(html, "POST") ||
			strings.Contains(html, "файл") ||
			strings.Contains(html, "форм") {
			nextData.Method = "POST"
		}

		re := regexp.MustCompile(`(?:<code>|<a href=")(/[^"<]*)`)
		if match := re.FindStringSubmatch(html); len(match) > 0 {
			nextData.Path = match[1]
		}

		nextData.Headers = parseTable(html, "следующие заголовки:", nextData.Headers)
		nextData.Cookies = parseTable(html, "выставлены cookie:", nextData.Cookies)
		nextData.Form = parseTable(html, "данные формы:", nextData.Form)
		nextData.Files = parseTable(html, "(использовать кодировку UTF8 без BOM):", nextData.Files)
		nextData.QueryParams = parseTable(html, " параметры запроса, указанные в таблице:", nextData.QueryParams)

		data = nextData
		fmt.Printf("Next: %s %s\n", data.Method, data.Path)
	}
}

func parseTable(html, sectionName string, target map[string]string) map[string]string {
	index := strings.Index(html, sectionName)
	if index == -1 {
		return target
	}

	section := html[index:]
	if endIdx := strings.Index(section, "</table>"); endIdx != -1 {
		section = section[:endIdx]
	}

	re := regexp.MustCompile(`(?s)<tr>\s*<td><code>(.*?)</code></td>\s*<td><code>(.*?)</code></td>\s*</tr>`)
	matches := re.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		target[match[1]] = match[2]
	}
	return target
}

func doRequest(data RequestData) string {
	conn, err := net.Dial("tcp", baseURL+":80")
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	if len(data.QueryParams) > 0 {
		params := url.Values{}
		for key, value := range data.QueryParams {
			params.Set(key, value)
		}
		data.Path += "?" + params.Encode()
	}

	var body []byte
	contentType := ""

	if data.Method == "POST" {
		if len(data.Files) > 0 {
			boundary := "-----------qwertyuiop123456789"
			contentType = "multipart/form-data; boundary=" + boundary
			var buffer bytes.Buffer
			for name, content := range data.Files {
				fmt.Fprintf(&buffer, "--%s\r\n", boundary)
				fmt.Fprintf(&buffer, "Content-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\n", name)
				fmt.Fprintf(&buffer, "Content-Type: text/plain\r\n\r\n%s\r\n", content)
			}
			fmt.Fprintf(&buffer, "--%s--\r\n", boundary)
			body = buffer.Bytes()
		} else {
			contentType = "application/x-www-form-urlencoded"
			values := url.Values{}
			for key, value := range data.Form {
				values.Set(key, value)
			}
			body = []byte(values.Encode())
		}
	}

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "%s %s HTTP/1.1\r\n", data.Method, data.Path)
	fmt.Fprintf(&buffer, "Host: %s\r\n", baseURL)

	for key, value := range data.Headers {
		fmt.Fprintf(&buffer, "%s: %s\r\n", key, value)
	}

	if len(data.Cookies) > 0 {
		fmt.Fprintf(&buffer, "Cookie: ")
		var cookieList []string
		for key, value := range data.Cookies {
			cookieList = append(cookieList, fmt.Sprintf("%s=%s", key, value))
		}
		fmt.Fprintf(&buffer, "%s\r\n", strings.Join(cookieList, "; "))
	}

	if len(body) > 0 {
		fmt.Fprintf(&buffer, "Content-Type: %s\r\n", contentType)
		fmt.Fprintf(&buffer, "Content-Length: %d\r\n", len(body))
	}
	fmt.Fprintf(&buffer, "Connection: close\r\n\r\n")

	conn.Write(buffer.Bytes())
	if len(body) > 0 {
		conn.Write(body)
	}

	response, _ := io.ReadAll(conn)
	return string(response)
}
