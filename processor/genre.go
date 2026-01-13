package processor

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/user/media-manager/logging"
	"github.com/user/media-manager/parser"
	"github.com/user/media-manager/utils"
)

// ProcessGenre检查并翻译NFO文件中的genre字段，返回是否修改了文件
func ProcessGenre(filePath string) (bool, error) {
	// 解析NFO文件
	nfo, err := parser.ParseNFO(filePath)
	if err != nil {
		return false, fmt.Errorf("处理genre时解析NFO文件失败: %w", err)
	}

	// 如果没有类型字段，返回错误
	if len(nfo.Genres) == 0 {
		logging.Warning("NFO文件中没有找到类型字段: %s", filePath)
		// 不返回错误，继续处理其他字段
		return false, nil
	}

	// 检查并翻译每个genre
	hasChanges := false
	for i, genre := range nfo.Genres {
		if !utils.IsSimplifiedChinese(genre) {
			translated := utils.TranslateGenre(genre)
			if translated != genre {
				logging.Info("将genre '%s' 翻译为 '%s'", genre, translated)
				nfo.Genres[i] = translated
				hasChanges = true
			}
		}
	}

	// 如果有变化，更新NFO文件
	if hasChanges {
		if err := updateGenreInFile(filePath, nfo.Genres); err != nil {
			return false, fmt.Errorf("更新genre字段失败: %w", err)
		}
		logging.Info("已更新NFO文件中的genre字段: %s", filePath)
	} else {
		logging.Info("NFO文件中的genre字段已经是简体中文: %s", filePath)
	}

	return hasChanges, nil
}

// updateGenreInFile更新NFO文件中的genre字段
func updateGenreInFile(filePath string, genres []string) error {
	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取NFO文件失败: %w", err)
	}

	// 将内容转换为字符串
	contentStr := string(content)

	// 移除所有旧的genre标签
	contentStr = removeGenreTags(contentStr)

	// 在合适的位置插入新的genre标签
	contentStr = insertGenreTags(contentStr, genres)

	// 写回文件
	if err := os.WriteFile(filePath, []byte(contentStr), 0644); err != nil {
		return fmt.Errorf("写入NFO文件失败: %w", err)
	}

	return nil
}

// removeGenreTags移除所有旧的genre标签
func removeGenreTags(content string) string {
	// 简单的字符串替换，移除所有<genre>和</genre>标签及其内容
	// 注意：这是一个简单的实现，可能需要根据实际的NFO文件结构调整
	content = regexp.MustCompile(`<genre>.*?</genre>`).ReplaceAllString(content, "")
	return content
}

// insertGenreTags在合适的位置插入新的genre标签
func insertGenreTags(content string, genres []string) string {
	// 找到插入位置，例如在<country>标签之后
	// 注意：这是一个简单的实现，可能需要根据实际的NFO文件结构调整
	insertPos := strings.Index(content, "</country>")
	if insertPos == -1 {
		// 如果没有找到country标签，尝试其他位置
		insertPos = strings.Index(content, "</year>")
		if insertPos == -1 {
			// 如果没有找到year标签，尝试其他位置
			insertPos = strings.Index(content, "</title>")
			if insertPos == -1 {
				// 如果都没有找到，返回原内容（不更新）
				return content
			}
		}
	}

	// 找到插入位置的行尾
	insertPos = strings.Index(content[insertPos:], "\n") + insertPos + 1

	// 构建新的genre标签
	var genreTags strings.Builder
	for _, genre := range genres {
		genreTags.WriteString(fmt.Sprintf("  <genre>%s</genre>\n", escapeXML(genre)))
	}

	// 插入新的genre标签
	content = content[:insertPos] + genreTags.String() + content[insertPos:]

	return content
}

// escapeXML转义XML特殊字符
func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
