package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	baseURL   = "hw1.alexbers.com"
	userToken = "e238c3c3730304c53a49f7ca0c04ce63"
	delay     = 1000
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

var stepNumberRe = regexp.MustCompile(`Шаг\s*#(\d+)`)
var tableRowRe = regexp.MustCompile(`(?s)<tr>\s*<td><code>(.*?)</code></td>\s*<td><code>(.*?)</code></td>\s*</tr>`)
var keyRe = regexp.MustCompile(`ключ: (\S*)`)

func main() {
	data := RequestData{
		Method:  "GET",
		Path:    "/",
		Cookies: map[string]string{"user": userToken},
	}

	for i := 1; ; i++ {
		html := doRequest(data)
		fmt.Printf("\nIteration: #%d\nStep:      #%s\n", i, getStepNumber(html))
		// fmt.Printf("\n[Server response]\n\n%s\n", html)
		
		if strings.Contains(html, "ключ") {
			fmt.Printf("\nKey: %s\n", html)
			break
		}

		data = makeNextData(html)
		// fmt.Printf("\n[Next]\nMethod:\n%s\nPath:\n%s\nCookies:\n%s\nHeaders:\n%s\nForm:\n%s\nFiles:\n%s\nQueryParams:\n%s\n", data.Method, data.Path, data.Cookies, data.Headers,
		// 	data.Form, data.Files, data.QueryParams)
		time.Sleep(delay * time.Millisecond)
	}
}

func getStepNumber(html string) string {
	match := stepNumberRe.FindStringSubmatch(html)
	if len(match) > 1 {
		return match[1]
	}
	return "?"
}

func makeNextData(html string) RequestData {
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

	re := regexp.MustCompile(`(?:<code>|<a href=["'])(/[^"'<]*)`)
	if match := re.FindStringSubmatch(html); len(match) > 0 {
		nextData.Path = match[1]
	}

	nextData.Headers = parseTable(html, "следующие заголовки:", nextData.Headers)
	nextData.Cookies = parseTable(html, "выставлены cookie:", nextData.Cookies)
	nextData.Form = parseTable(html, "данные формы:", nextData.Form)
	nextData.Files = parseTable(html, "Имя файла", nextData.Files)
	nextData.QueryParams = parseTable(html, "параметры запроса, указанные в таблице:", nextData.QueryParams)

	return nextData
}

func parseTable(html, sectionName string, target map[string]string) map[string]string {
	section := findSection(html, sectionName)
	matches := tableRowRe.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		target[match[1]] = match[2]
	}
	return target
}

func findSection(html, sectionName string) string {
    index := strings.Index(html, sectionName)
    if index == -1 {
        return ""
    }
    section := html[index:]
    endIndex := strings.Index(section, "</table>")
    if endIndex == -1 {
        return section
    }
    return section[:endIndex]
}


func doRequest(data RequestData) string {
	conn, err := net.Dial("tcp", baseURL+":80")
	if err != nil {
		fmt.Printf("[Error] %v", err)
		return ""
	}
	defer conn.Close()

	fullPath := buildPath(data.Path, data.QueryParams)
	body, contentType := buildBody(data)
	headerPart := buildHeaders(data, fullPath, len(body), contentType)

	// fmt.Printf("\n[Request header]\n\n%s\n", headerPart)
	conn.Write(headerPart)
	if len(body) > 0 {
		conn.Write(body)
		// fmt.Printf("\n[Request body]\n\n%s\n", body)
	}

	response, _ := io.ReadAll(conn)
	return string(response)
}

func buildPath(path string, queryParams map[string]string) string {
	if len(queryParams) == 0 {
		return path
	}
	params := url.Values{}
	for key, value := range queryParams {
		params.Set(key, value)
	}
	return path + "?" + params.Encode()
}

func buildBody(data RequestData) ([]byte, string) {
	if data.Method != "POST" {
		return nil, ""
	}

	if len(data.Files) > 0 {

		// const boundary = "-----------qwertyuiop123456789"
		// var buffer bytes.Buffer
		// for name, content := range data.Files {
		// 	fmt.Fprintf(&buffer, "--%s\r\n", boundary)
		// 	fmt.Fprintf(&buffer, "Content-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\n", name)
		// 	fmt.Fprintf(&buffer, "Content-Type: text/plain\r\n\r\n%s\r\n", content)
		// }
		// fmt.Fprintf(&buffer, "--%s--\r\n", boundary)
		// return buffer.Bytes(), "multipart/form-data; boundary=" + boundary

		var buffer bytes.Buffer
		writer := multipart.NewWriter(&buffer)
		for name, content := range data.Files {
			part, err := writer.CreateFormFile("file", name)
			if err != nil {
				continue
			}
			part.Write([]byte(content))
		}

		writer.Close()
		return buffer.Bytes(), writer.FormDataContentType()
	}

	if len(data.Form) > 0 {
		values := url.Values{}
		for key, value := range data.Form {
			values.Set(key, value)
		}

		return []byte(values.Encode()), "application/x-www-form-urlencoded"
	}

	return nil, ""
}

func buildHeaders(data RequestData, path string, bodyLen int, contentType string) []byte {
	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, "%s %s HTTP/1.1\r\n", data.Method, path)
	fmt.Fprintf(&buffer, "Host: %s\r\n", baseURL)

	for key, values := range data.Headers {
		fmt.Fprintf(&buffer, "%s: %s\r\n", key, values)
	}

	if len(data.Cookies) > 0 {
		var cookies []string
		for key, value := range data.Cookies {
			cookies = append(cookies, fmt.Sprintf("%s=%s", key, value))
		}
		fmt.Fprintf(&buffer, "Cookie: %s\r\n", strings.Join(cookies, "; "))
	}

	if bodyLen > 0 {
		fmt.Fprintf(&buffer, "Content-Type: %s\r\n", contentType)
		fmt.Fprintf(&buffer, "Content-Length: %d\r\n", bodyLen)
	}

	fmt.Fprintf(&buffer, "Connection: close\r\n\r\n")
	return buffer.Bytes()
}
