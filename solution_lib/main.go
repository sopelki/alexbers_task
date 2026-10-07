package main

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	baseURL   = "http://hw1.alexbers.com"
	userToken = "e238c3c3730304c53a49f7ca0c04ce63"
	delay     = 200
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
		Method: "GET",
		Path:   "/",
	}

	client := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(0).
		SetHeader("Connection", "close").
		SetDebug(false)

	for i := 1; ; i++ {
		html := doRequest(data, client)
		fmt.Printf("\n--- Step #%d (%s) ---\n", i, getStepNumber(html))

		// fmt.Printf("\n[Server response]\n\n%s\n", html)

		if strings.Contains(html, "ключ") {
			fmt.Printf("\n[Succes]\n\n%s\n", html)
			break
		}

		data = makeNextData(html)
		// fmt.Printf("\n[Next]\nMethod:\n%s\nPath:\n%s\nCookies:\n%s\nHeaders:\n%s\nForm:\n%s\nFiles:\n%s\nQueryParams:\n%s\n", data.Method, data.Path, data.Cookies, data.Headers,
		// 	data.Form, data.Files, data.QueryParams)
		time.Sleep(delay * time.Millisecond)
	}
}

func getStepNumber(html string) string {
	re := regexp.MustCompile(`Шаг\s*#(\d+)`)
	match := re.FindStringSubmatch(html)
	if len(match) > 1 {
		return match[1]
	}
	return "?"
}

func makeNextData(html string) RequestData {
	nextData := RequestData{
		Method:      "GET",
		Cookies:     make(map[string]string),
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
	index := strings.Index(html, sectionName)
	if index == -1 {
		return target
	}

	section := html[index:]
	if endIndex := strings.Index(section, "</table>"); endIndex != -1 {
		section = section[:endIndex]
	}

	re := regexp.MustCompile(`(?s)<tr>\s*<td><code>(.*?)</code></td>\s*<td><code>(.*?)</code></td>\s*</tr>`)
	matches := re.FindAllStringSubmatch(section, -1)
	for _, match := range matches {
		target[match[1]] = match[2]
	}
	return target
}

func doRequest(data RequestData, client *resty.Client) string {
	cookies := make([]*http.Cookie, 0, len(data.Cookies))
	for name, value := range data.Cookies {
		cookies = append(cookies, &http.Cookie{Name: name, Value: value})
	}

	req := client.R().
		SetQueryParams(data.QueryParams).
		SetHeaders(data.Headers).
		SetCookies(cookies).
		SetHeader("Expect", "")

	if _, ok := data.Cookies["user"]; !ok {
		req.SetCookie(&http.Cookie{Name: "user", Value: userToken})
	}

	if data.Method == "POST" {
		if len(data.Files) > 0 {
			fields := make([]*resty.MultipartField, 0, len(data.Files))
			for name, content := range data.Files {
				fields = append(fields, &resty.MultipartField{
					Param:       "file",
					FileName:    name,
					ContentType: "text/plain",
					Reader:      strings.NewReader(content),
				})
			}
			req.SetMultipartFields(fields...)
		} else if len(data.Form) > 0 {
			req.SetFormData(data.Form)
		}
	}

	var response *resty.Response
	var err error

	switch data.Method {
	case "GET":
		response, err = req.Get(data.Path)
	case "POST":
		response, err = req.Post(data.Path)
	}

	if err != nil {
		fmt.Printf("\n[Error] %v\n", err)
		return ""
	}

	// fmt.Print(req.RawRequest)
	return response.String()
}
