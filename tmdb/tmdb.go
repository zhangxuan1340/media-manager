package tmdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/user/media-manager/config"
)

// 国家代码到中文名称的映射表
var countryCodeToChinese = map[string]string{
	"US": "美国",
	"CN": "中国",
	"HK": "中国香港",
	"TW": "中国台湾",
	"JP": "日本",
	"KR": "韩国",
	"GB": "英国",
	"CA": "加拿大",
	"FR": "法国",
	"DE": "德国",
	"IT": "意大利",
	"ES": "西班牙",
	"AU": "澳大利亚",
	"IN": "印度",
	"RU": "俄罗斯",
	"BR": "巴西",
	"MX": "墨西哥",
	"TH": "泰国",
	"ID": "印度尼西亚",
	"MY": "马来西亚",
	"SG": "新加坡",
	"PH": "菲律宾",
	"VN": "越南",
	"AE": "阿联酋",
	"AR": "阿根廷",
	"AT": "奥地利",
	"BE": "比利时",
	"CH": "瑞士",
	"CL": "智利",
	"CO": "哥伦比亚",
	"CZ": "捷克",
	"DK": "丹麦",
	"EG": "埃及",
	"FI": "芬兰",
	"GR": "希腊",
	"HU": "匈牙利",
	"IE": "爱尔兰",
	"IL": "以色列",
	"IS": "冰岛",
	"KE": "肯尼亚",
	"NL": "荷兰",
	"NO": "挪威",
	"NZ": "新西兰",
	"PE": "秘鲁",
	"PL": "波兰",
	"PT": "葡萄牙",
	"RO": "罗马尼亚",
	"SE": "瑞典",
	"SA": "沙特阿拉伯",
	"TR": "土耳其",
	"ZA": "南非",
	"BD": "孟加拉国",
	"BG": "保加利亚",
	"BO": "玻利维亚",
	"BT": "不丹",
	"BY": "白俄罗斯",
	"CU": "古巴",
	"DO": "多米尼加共和国",
	"EC": "厄瓜多尔",
	"EE": "爱沙尼亚",
	"GE": "格鲁吉亚",
	"GH": "加纳",
	"GT": "危地马拉",
	"HN": "洪都拉斯",
	"HR": "克罗地亚",
	"HT": "海地",
	"IQ": "伊拉克",
	"IR": "伊朗",
	"JO": "约旦",
	"KH": "柬埔寨",
	"KM": "科摩罗",
	"KP": "朝鲜",
	"KW": "科威特",
	"KY": "开曼群岛",
	"KZ": "哈萨克斯坦",
	"LB": "黎巴嫩",
	"LK": "斯里兰卡",
	"LT": "立陶宛",
	"LU": "卢森堡",
	"LV": "拉脱维亚",
	"MA": "摩洛哥",
	"MD": "摩尔多瓦",
	"ME": "黑山",
	"MK": "北马其顿",
	"MN": "蒙古",
	"MT": "马耳他",
	"MW": "马拉维",
	"NE": "尼日尔",
	"NG": "尼日利亚",
	"NI": "尼加拉瓜",
	"OM": "阿曼",
	"PA": "巴拿马",
	"PG": "巴布亚新几内亚",
	"PK": "巴基斯坦",
	"PR": "波多黎各",
	"PS": "巴勒斯坦",
	"PY": "巴拉圭",
	"QA": "卡塔尔",
	"RS": "塞尔维亚",
	"RW": "卢旺达",
	"SV": "萨尔瓦多",
	"SY": "叙利亚",
	"TZ": "坦桑尼亚",
	"UA": "乌克兰",
	"UG": "乌干达",
	"UY": "乌拉圭",
	"UZ": "乌兹别克斯坦",
	"VE": "委内瑞拉",
	"YE": "也门",
	"ZM": "赞比亚",
	"ZW": "津巴布韦",
}

// TMDBResponse 表示TMDB API的响应结构
type TMDBResponse struct {
	ProductionCountries []ProductionCountry `json:"production_countries"`
	OriginalLanguage    string              `json:"original_language"`
}

// TVShowResponse 表示TMDB API返回的电视剧信息
type TVShowResponse struct {
	NumberOfSeasons     int                 `json:"number_of_seasons"`
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

	// 构建API URL
	var baseURL string
	if cfg.UseTMDBOrg {
		baseURL = "https://api.tmdb.org/3/" // 使用tmdb.org
	} else {
		baseURL = "https://api.themoviedb.org/3/" // 使用themoviedb.org
	}

	endpoint := "movie/"
	if isTVShow {
		endpoint = "tv/"
	}

	var apiURL string
	if apiKey == "" {
		// 没有API密钥时，尝试不使用密钥访问
		apiURL = fmt.Sprintf("%s%s%s?language=zh-CN", baseURL, endpoint, tmdbID)
	} else {
		// 有API密钥时，使用密钥访问
		apiURL = fmt.Sprintf("%s%s%s?api_key=%s&language=zh-CN", baseURL, endpoint, tmdbID, apiKey)
	}

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
	var countries []string
	if isTVShow {
		var tvResp TVShowResponse
		if err := json.Unmarshal(body, &tvResp); err != nil {
			return nil, fmt.Errorf("解析TMDB API响应失败: %w", err)
		}
		for _, country := range tvResp.ProductionCountries {
			// 使用国家代码查找中文名称
			if chineseName, exists := countryCodeToChinese[country.ISO3166_1]; exists {
				countries = append(countries, chineseName)
			} else {
				// 如果没有找到对应的中文名称，使用API返回的名称
				countries = append(countries, country.Name)
			}
		}
	} else {
		var tmdbResp TMDBResponse
		if err := json.Unmarshal(body, &tmdbResp); err != nil {
			return nil, fmt.Errorf("解析TMDB API响应失败: %w", err)
		}
		for _, country := range tmdbResp.ProductionCountries {
			// 使用国家代码查找中文名称
			if chineseName, exists := countryCodeToChinese[country.ISO3166_1]; exists {
				countries = append(countries, chineseName)
			} else {
				// 如果没有找到对应的中文名称，使用API返回的名称
				countries = append(countries, country.Name)
			}
		}
	}

	return countries, nil
}

// GetOriginalLanguage 获取原始语言
func GetOriginalLanguage(tmdbID string, isTVShow bool) (string, error) {
	// 加载配置
	cfg := config.LoadConfig()
	apiKey := cfg.TMDBApiKey

	// 构建API URL
	var baseURL string
	if cfg.UseTMDBOrg {
		baseURL = "https://api.tmdb.org/3/" // 使用tmdb.org
	} else {
		baseURL = "https://api.themoviedb.org/3/" // 使用themoviedb.org
	}

	endpoint := "movie/"
	if isTVShow {
		endpoint = "tv/"
	}

	var apiURL string
	if apiKey == "" {
		// 没有API密钥时，尝试不使用密钥访问
		apiURL = fmt.Sprintf("%s%s%s?language=zh-CN", baseURL, endpoint, tmdbID)
	} else {
		// 有API密钥时，使用密钥访问
		apiURL = fmt.Sprintf("%s%s%s?api_key=%s&language=zh-CN", baseURL, endpoint, tmdbID, apiKey)
	}

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
	var originalLanguage string
	if isTVShow {
		var tvResp TVShowResponse
		if err := json.Unmarshal(body, &tvResp); err != nil {
			return "", fmt.Errorf("解析TMDB API响应失败: %w", err)
		}
		originalLanguage = tvResp.OriginalLanguage
	} else {
		var tmdbResp TMDBResponse
		if err := json.Unmarshal(body, &tmdbResp); err != nil {
			return "", fmt.Errorf("解析TMDB API响应失败: %w", err)
		}
		originalLanguage = tmdbResp.OriginalLanguage
	}

	return originalLanguage, nil
}

// GetTVShowSeasons 获取电视剧的总季数
func GetTVShowSeasons(tmdbID string) (int, error) {
	// 加载配置
	cfg := config.LoadConfig()
	apiKey := cfg.TMDBApiKey

	// 构建API URL
	var baseURL string
	if cfg.UseTMDBOrg {
		baseURL = "https://api.tmdb.org/3/" // 使用tmdb.org
	} else {
		baseURL = "https://api.themoviedb.org/3/" // 使用themoviedb.org
	}

	endpoint := "tv/"
	var apiURL string
	if apiKey == "" {
		// 没有API密钥时，尝试不使用密钥访问
		apiURL = fmt.Sprintf("%s%s%s?language=zh-CN", baseURL, endpoint, tmdbID)
	} else {
		// 有API密钥时，使用密钥访问
		apiURL = fmt.Sprintf("%s%s%s?api_key=%s&language=zh-CN", baseURL, endpoint, tmdbID, apiKey)
	}

	// 发送请求
	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, fmt.Errorf("TMDB API请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("TMDB API返回错误状态码: %d", resp.StatusCode)
	}

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取TMDB API响应失败: %w", err)
	}

	// 解析JSON
	var tvResp TVShowResponse
	if err := json.Unmarshal(body, &tvResp); err != nil {
		return 0, fmt.Errorf("解析TMDB API响应失败: %w", err)
	}

	return tvResp.NumberOfSeasons, nil
}
