package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/user/media-manager/config"
)

// TMDBResponse 表示TMDB API的响应结构
type TMDBResponse struct {
	ProductionCountries []ProductionCountry `json:"production_countries"`
	OriginalLanguage    string              `json:"original_language"`
}

// ProductionCountry 表示制作国家信息
type ProductionCountry struct {
	ISO3166_1 string `json:"iso_3166_1"`
	Name      string `json:"name"`
}

// GetProductionCountries 获取电影或电视剧的制作国家信息
func GetProductionCountries(tmdbID string, isTVShow bool) ([]string, error) {
	// 加载配置
	cfg := config.LoadConfig()
	apiKey := cfg.TMDBApiKey

	if apiKey == "" {
		return nil, fmt.Errorf("TMDB API密钥未配置")
	}

	// 构建API URL
	baseURL := "https://api.themoviedb.org/3/"
	endpoint := "movie/"
	if isTVShow {
		endpoint = "tv/"
	}

	apiURL := fmt.Sprintf("%s%s%s?api_key=%s&language=zh-CN", baseURL, endpoint, tmdbID, apiKey)

	// 发送请求
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("TMDB API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("TMDB API返回错误状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取TMDB API响应失败: %w", err)
	}

	// 解析JSON
	var tmdbResp TMDBResponse
	if err := json.Unmarshal(body, &tmdbResp); err != nil {
		return nil, fmt.Errorf("解析TMDB API响应失败: %w", err)
	}

	// 提取制作国家名称
	var countries []string
	for _, country := range tmdbResp.ProductionCountries {
		countries = append(countries, country.Name)
	}

	return countries, nil
}

// GetOriginalLanguage 获取原始语言
func GetOriginalLanguage(tmdbID string, isTVShow bool) (string, error) {
	// 加载配置
	cfg := config.LoadConfig()
	apiKey := cfg.TMDBApiKey

	if apiKey == "" {
		return "", fmt.Errorf("TMDB API密钥未配置")
	}

	// 构建API URL
	baseURL := "https://api.themoviedb.org/3/"
	endpoint := "movie/"
	if isTVShow {
		endpoint = "tv/"
	}

	apiURL := fmt.Sprintf("%s%s%s?api_key=%s", baseURL, endpoint, tmdbID, apiKey)

	// 发送请求
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("TMDB API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("TMDB API返回错误状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取TMDB API响应失败: %w", err)
	}

	// 解析JSON
	var tmdbResp TMDBResponse
	if err := json.Unmarshal(body, &tmdbResp); err != nil {
		return "", fmt.Errorf("解析TMDB API响应失败: %w", err)
	}

	return tmdbResp.OriginalLanguage, nil
}
