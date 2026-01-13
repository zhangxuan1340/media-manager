package processor

import (
	"fmt"
	"os"
	"time"

	"github.com/user/media-manager/logging"
	"github.com/user/media-manager/parser"
	"github.com/user/media-manager/utils"
)

// ActorReport表示演员检查报告的结构
type ActorReport struct {
	FileName string
	Actors   []ActorIssue
}

// ActorIssue表示单个演员的问题
type ActorIssue struct {
	Name  string
	Role  string
	Issue string
}

// ProcessActor检查NFO文件中的演员名称是否为中文
func ProcessActor(filePath string) (*ActorReport, error) {
	// 解析NFO文件
	nfo, err := parser.ParseNFO(filePath)
	if err != nil {
		return nil, fmt.Errorf("处理actor时解析NFO文件失败: %w", err)
	}

	// 创建报告
	report := &ActorReport{
		FileName: filePath,
		Actors:   []ActorIssue{},
	}

	// 检查每个演员的名称
	for _, actor := range nfo.Actors {
		if !utils.IsChinese(actor.Name) {
			issue := ActorIssue{
				Name:  actor.Name,
				Role:  actor.Role,
				Issue: "演员名称非中文",
			}
			report.Actors = append(report.Actors, issue)
		}
	}

	// 如果有非中文演员，生成报告
	if len(report.Actors) > 0 {
		if err := generateActorReport(report); err != nil {
			return nil, fmt.Errorf("生成演员报告失败: %w", err)
		}
		logging.Info("发现非中文演员名称，已生成报告: %s", filePath)
	} else {
		logging.Info("所有演员名称都是中文: %s", filePath)
	}

	return report, nil
}

// generateActorReport生成演员检查报告文件
func generateActorReport(report *ActorReport) error {
	// 创建报告目录，使用临时目录
	reportDir := "/tmp/media-manager/reports"
	if err := os.MkdirAll(reportDir, 0755); err != nil {
		return fmt.Errorf("创建报告目录失败: %w", err)
	}

	// 生成报告文件名（只包含年月日）
	dateStr := time.Now().Format("20060102")
	reportFileName := fmt.Sprintf("%s/actor_report_%s.txt", reportDir, dateStr)

	// 检查文件是否存在
	file, err := os.OpenFile(reportFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开报告文件失败: %w", err)
	}
	defer file.Close()

	// 写入报告内容（追加模式）
	fmt.Fprintf(file, "\n\n-------------------- 新检查记录 --------------------\n")
	fmt.Fprintf(file, "检查时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "检查文件: %s\n", report.FileName)

	if len(report.Actors) > 0 {
		fmt.Fprintf(file, "发现以下非中文演员名称:\n")
		fmt.Fprintf(file, "%-30s %-30s %-20s\n", "演员名称", "角色", "问题")
		fmt.Fprintf(file, "%-30s %-30s %-20s\n", "--------", "--------", "--------")

		for _, actor := range report.Actors {
			fmt.Fprintf(file, "%-30s %-30s %-20s\n", actor.Name, actor.Role, actor.Issue)
		}
	} else {
		fmt.Fprintf(file, "所有演员名称都是中文。\n")
	}

	logging.Info("演员检查报告已生成: %s", reportFileName)

	return nil
}
