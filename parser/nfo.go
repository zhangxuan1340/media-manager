package parser

import (
	"encoding/xml"
	"fmt"
	"os"
)

// NFO表示NFO文件的结构
type NFO struct {
	XMLName       xml.Name // 根标签，动态设置为movie或tvshow
	Title         string   `xml:"title"`
	OriginalTitle string   `xml:"originaltitle"`
	Year          string   `xml:"year"`
	Country       []string `xml:"country"`
	Genres        []string `xml:"genre"`
	Actors        []Actor  `xml:"actor"`
	Runtime       string   `xml:"runtime"`
	Plot          string   `xml:"plot"`
	IMDbID        string   `xml:"id" xml:"imdbid"`
	TMDbID        string   `xml:"tmdbid"`
	Season        string   `xml:"season"`
	Episode       string   `xml:"episode"`
	Director      string   `xml:"director"`
	Writer        string   `xml:"writer"`
	Rating        string   `xml:"rating"`
	// 其他可能需要的字段
}

// Actor表示演员信息
type Actor struct {
	Name string `xml:"name"`
	Role string `xml:"role"`
}

// ParseNFO解析指定路径的NFO文件
func ParseNFO(filePath string) (*NFO, error) {
	// 打开NFO文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法打开NFO文件: %w", err)
	}
	defer file.Close()

	// 创建XML解码器
	decoder := xml.NewDecoder(file)

	// 跳过XML声明
	for {
		token, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("无法解析NFO文件: %w", err)
		}
		if startElement, ok := token.(xml.StartElement); ok {
			// 检查根标签类型
			if startElement.Name.Local == "movie" || startElement.Name.Local == "tvshow" {
				// 创建NFO结构体并设置根标签
				var nfo NFO
				nfo.XMLName = startElement.Name

				// 解析剩余内容
				if err := decoder.DecodeElement(&nfo, &startElement); err != nil {
					return nil, fmt.Errorf("无法解析NFO文件: %w", err)
				}

				return &nfo, nil
			}
			return nil, fmt.Errorf("不支持的NFO文件类型: %s", startElement.Name.Local)
		}
	}
}

// IsTVShow判断是否为电视剧（根据XML根标签）
func (n *NFO) IsTVShow() bool {
	// 根据XML根标签判断：如果是tvshow则为电视剧，否则为电影
	return n.XMLName.Local == "tvshow"
}

// GetFullTitle获取完整的影片标题
func (n *NFO) GetFullTitle() string {
	if n.OriginalTitle != "" {
		return fmt.Sprintf("%s (%s)", n.Title, n.Year)
	}
	return fmt.Sprintf("%s (%s)", n.Title, n.Year)
}
