package geecache

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// 客户端
type httpClient struct {
	baseURL string //TODO baseURL最后一个字符一定要是"/"
}

func NewHttpClient(baseURL string) PeerClient {
	return &httpClient{
		baseURL: baseURL,
	}
}

// baseURL 表示将要访问的远程节点的地址，例如 http://example.com/_geecache/。
// 使用 http.Get() 方式获取返回值，并转换为 []bytes 类型。
func (h *httpClient) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key),
	)
	res, err := http.Get(u)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}
