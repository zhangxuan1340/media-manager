package utils

import (
	"regexp"
	"strings"
)

// IsSimplifiedChinese检查字符串是否包含简体中文
func IsSimplifiedChinese(s string) bool {
	// 检查字符串是否至少包含一个中文字符
	containsChineseReg := regexp.MustCompile(`[\p{Han}]`)
	return containsChineseReg.MatchString(s)
}

// IsChinese检查字符串是否包含中文（简体或繁体）
func IsChinese(s string) bool {
	// 使用Unicode属性来检查中文字符
	reg := regexp.MustCompile(`[\p{Han}]`)
	return reg.MatchString(s)
}

// GenreMap定义类型映射规则
var GenreMap = map[string]string{
	"Action":          "动作",
	"Adventure":       "冒险",
	"Animation":       "动画",
	"Comedy":          "喜剧",
	"Crime":           "犯罪",
	"Documentary":     "纪录片",
	"Drama":           "剧情",
	"Family":          "家庭",
	"Fantasy":         "奇幻",
	"Horror":          "恐怖",
	"Mystery":         "悬疑",
	"Romance":         "爱情",
	"Science Fiction": "科幻",
	"Thriller":        "惊悚",
	"War":             "战争",
	"Western":         "西部",
	"Biography":       "传记",
	"History":         "历史",
	"Music":           "音乐",
	"Musical":         "歌舞",
	"Sport":           "体育",
	"Talk Show":       "脱口秀",
	"Variety Show":    "综艺节目",
	"News":            "新闻",
	"Reality TV":      "真人秀",
	"Game-Show":       "游戏节目",
	"Game Show":       "游戏节目",
	"Variety":         "综艺节目",
	"Film-Noir":       "黑色电影",
	"Short":           "短片",
	"TV Movie":        "电视电影",
	"Competition":     "竞赛节目",
	"Talent Show":     "选秀节目",
	// 可以根据需要添加更多映射
}

// TranslateGenre将英文类型翻译为简体中文
func TranslateGenre(genre string) string {
	// 去除首尾空格并转换为小写
	genre = strings.TrimSpace(genre)
	lowerGenre := strings.ToLower(genre)

	// 检查是否已经是简体中文
	if IsSimplifiedChinese(genre) {
		return genre
	}

	// 查找映射
	for en, cn := range GenreMap {
		if strings.ToLower(en) == lowerGenre {
			return cn
		}
	}

	// 如果没有找到映射，返回原类型
	return genre
}
