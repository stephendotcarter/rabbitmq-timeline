package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	BuildVersion string
	BuildDate    string
)

type Node struct {
	FileName        string
	Name            string
	HomeDir         string
	CookieHash      []string
	DatabaseDir     string
	VersionRabbitMQ []string
	VersionErlang   []string
}

type Report struct {
	Message  string
	Severity string
}

type LogMessage struct {
	Node     int
	DateTime string
	Severity string
	Pid      string
	Message  []string
	Reports  []Report
	Order    int
}

func NewNode(logPath string) Node {
	var node Node
	node.FileName = logPath
	return node
}

func NewNodes(inputFiles []string) []Node {
	var newNodes []Node
	for _, logPath := range inputFiles {
		_, err := os.Stat(logPath)
		if err != nil {
			log.Printf("Cannot access \"%s\":\n%s", logPath, err)
			os.Exit(1)
		}
		newNode := NewNode(logPath)
		newNodes = append(newNodes, newNode)
	}
	return newNodes
}

func NewLogMessageFromLine(line []string) LogMessage {
	var newLogMessage LogMessage

	newLogMessage.DateTime = line[1] + " " + line[2]
	newLogMessage.Severity = line[3]
	newLogMessage.Pid = line[4]
	newLogMessage.Message = []string{line[5]}

	return newLogMessage
}

func RenderLogMessageRow(message LogMessage) string {
	var html bytes.Buffer
	html.WriteString(fmt.Sprintf("<tr>"))
	html.WriteString(fmt.Sprintf("<td>%d</td>", message.Node))
	html.WriteString(fmt.Sprintf("<td class=\"nowrap\">%s</td>", message.DateTime))
	html.WriteString(fmt.Sprintf("<td>%s</td>", message.Severity))
	html.WriteString(fmt.Sprintf("<td>%s</td>", message.Pid))
	html.WriteString(fmt.Sprintf("<td><pre>%s</pre></td>", strings.Join(message.Message[:], "\n")))
	html.WriteString(fmt.Sprintf("</tr>"))
	return html.String()
}

func RenderNodeHeader(node Node) string {
	var html bytes.Buffer
	html.WriteString(fmt.Sprintf("<td><div>"))
	html.WriteString(fmt.Sprintf("<strong>Filename:</strong><div class=\"indent\">%s</div>", node.FileName))
	html.WriteString(fmt.Sprintf("<strong>Name:</strong><div class=\"indent\">%s</div>", node.Name))
	html.WriteString(fmt.Sprintf("<strong>RabbitMQ:</strong><div class=\"indent\">%s</div>", strings.Join(node.VersionRabbitMQ, "<br>")))
	html.WriteString(fmt.Sprintf("<strong>Erlang:</strong><div class=\"indent\">%s</div>", strings.Join(node.VersionErlang, "<br>")))
	html.WriteString(fmt.Sprintf("<strong>Cookie Hash:</strong><div class=\"indent\">%s</div>", strings.Join(node.CookieHash, "<br>")))
	html.WriteString(fmt.Sprintf("</div></td>"))
	return html.String()
}

func RemoveDuplicatesFromSlice(s []string) []string {
	m := make(map[string]bool)
	for _, item := range s {
		if _, ok := m[item]; ok {
			continue
		} else {
			m[item] = true
		}
	}

	var result []string
	for item, _ := range m {
		result = append(result, item)
	}
	return result
}

func checkLogMessageForReport(logMessage *LogMessage, nodes []Node) {
	for _, message := range logMessage.Message {
		if strings.Contains(message, "node           :") {
			parts := strings.Split(message, " : ")
			nodes[logMessage.Node].Name = logMessage.DateTime + " " + parts[1]
		}
		if strings.Contains(message, "cookie hash    :") {
			parts := strings.Split(message, " : ")
			nodes[logMessage.Node].CookieHash = append(nodes[logMessage.Node].CookieHash, logMessage.DateTime+" "+parts[1])
		}
		if strings.Contains(message, "Starting RabbitMQ ") {
			parts := strings.Split(message, " ")
			nodes[logMessage.Node].VersionRabbitMQ = append(nodes[logMessage.Node].VersionRabbitMQ, logMessage.DateTime+" "+parts[3])
			nodes[logMessage.Node].VersionErlang = append(nodes[logMessage.Node].VersionErlang, logMessage.DateTime+" "+parts[6])
			logMessage.Reports = append(logMessage.Reports, Report{
				"RabbitMQ is starting",
				"info",
			})
		}
		if strings.Contains(message, "Assuming we need to join an existing cluster or initialise from scratch...") {
			logMessage.Reports = append(logMessage.Reports, Report{
				"Mnesia directory was empty",
				"warning",
			})
		}
		if strings.Contains(message, "RabbitMQ is asked to stop...") {
			logMessage.Reports = append(logMessage.Reports, Report{
				"Stopped via \"rabbitmqctl\" or internally",
				"warning",
			})
		}
		if strings.Contains(message, "SIGTERM received - shutting down") {
			logMessage.Reports = append(logMessage.Reports, Report{
				"Stopped via \"SIGTERM\"",
				"warning",
			})
		}
		if strings.Contains(message, "Memory high watermark set to ") {
			parts := strings.Split(message, " ")
			logMessage.Reports = append(logMessage.Reports, Report{
				fmt.Sprintf("Memory limit: %s", parts[5]),
				"info",
			})
		}
		if strings.Contains(message, "Disk free limit set to ") {
			parts := strings.Split(message, " ")
			logMessage.Reports = append(logMessage.Reports, Report{
				fmt.Sprintf("Disk free limit: %s", parts[5]),
				"info",
			})
		}
		if strings.Contains(message, "Limiting to approx ") {
			parts := strings.Split(message, " ")
			logMessage.Reports = append(logMessage.Reports, Report{
				fmt.Sprintf("File handle limit: %s", parts[3]),
				"info",
			})
		}
		if strings.Contains(message, "Free disk space is sufficient.") {
			logMessage.Reports = append(logMessage.Reports, Report{
				message,
				"info",
			})
		}
		if strings.Contains(message, "Free disk space is insufficient.") {
			logMessage.Reports = append(logMessage.Reports, Report{
				message,
				"error",
			})
		}
		if strings.Contains(message, "disk resource limit alarm set on node ") {
			logMessage.Reports = append(logMessage.Reports, Report{
				message,
				"error",
			})
		}
		if strings.Contains(message, " down: net_tick_timeout") {
			logMessage.Reports = append(logMessage.Reports, Report{
				message,
				"error",
			})
		}
	}
}

func generateReportHTML(logTable map[string][][]*LogMessage, logDateTimes []string, nodes []Node) string {
	var html bytes.Buffer

	htmlStyle := `
	<style>
	html, td {
		font-family: monospace;
		font-size: 12px;
	}
	body {
		margin: 10px;
	}
	pre {
		margin: 0px;
	}
	h1, h2, h3, h4 {
		margin: 0px;
	}
	td {
		vertical-align: top;
	}
	table {
		border-top: 1px solid #EEE;
		border-left: 1px solid #EEE;
	}
	td, th {
		border-bottom: 1px solid #EEE;
		border-right: 1px solid #EEE;
		padding: 0px;
		vertical-align: top;
	}
	td > div {
		padding: 3px;
	}
	.indent {
		margin-left:15px;
	}
	.nowrap {
		white-space: nowrap;
	}
	.header {
		color: #FFF;
		background-color: #171717;
	}
	.header td {
		padding: 5px;
		font-family: Arial;
	}
	.prewrap {
		white-space: pre-wrap;
	}
	.s_info {
		background-color: #FFFFFF;
	}
	.s_notice {
		background-color: #4DA6FF;
	}
	.s_warning {
		background-color: #FFA64D;
	}
	.s_error {
		background-color: #FF4D4D;
	}
	.s_report {
		background-color: #4DFF4D;
	}
	</style>`

	html.WriteString(fmt.Sprintf(htmlStyle))
	html.WriteString(fmt.Sprintf("<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\">\n"))
	html.WriteString(fmt.Sprintf("<thead>\n"))
	html.WriteString(fmt.Sprintf("<tr class=\"header\">"))
	html.WriteString(fmt.Sprintf("<td><h3>Summary</h3></td>\n"))
	for _, node := range nodes {
		html.WriteString(fmt.Sprintf("<td><h3>%s</h3></td>\n", filepath.Base(node.FileName)))
	}
	html.WriteString(fmt.Sprintf("</tr>"))
	html.WriteString(fmt.Sprintf("<tr>"))
	html.WriteString(fmt.Sprintf("<th></td>"))
	for _, node := range nodes {
		html.WriteString(RenderNodeHeader(node))
	}
	html.WriteString(fmt.Sprintf("</tr>"))
	html.WriteString(fmt.Sprintf("</thead>\n"))

	html.WriteString(fmt.Sprintf("<tbody>\n"))
	html.WriteString(fmt.Sprintf("<tr class=\"header\">"))
	html.WriteString(fmt.Sprintf("<td><h3>Timeline</h></td>\n"))
	for _, node := range nodes {
		html.WriteString(fmt.Sprintf("<td><h3>%s</h3></td>\n", filepath.Base(node.FileName)))
	}
	html.WriteString(fmt.Sprintf("</tr>\n"))

	for _, logDateTime := range logDateTimes {
		html.WriteString(fmt.Sprintf("<tr>\n"))
		html.WriteString(fmt.Sprintf("<td class=\"nowrap\"><div>%s</div></td>\n", logDateTime))

		for _, nodeLogs := range logTable[logDateTime] {
			html.WriteString(fmt.Sprintf("<td>\n"))
			for _, nodeLog := range nodeLogs {
				html.WriteString(fmt.Sprintf("<div class=\"prewrap s_%s\"><strong>[%s]</strong> %s</div>", nodeLog.Severity, nodeLog.Severity, strings.Join(nodeLog.Message[:], "\n")))
				for _, nodeReport := range nodeLog.Reports {
					html.WriteString(fmt.Sprintf("<div class=\"prewrap s_report\"><strong>=REPORT=</strong> %s</div>", nodeReport.Message))
				}
			}
			html.WriteString(fmt.Sprintf("</td>\n"))
		}
		html.WriteString(fmt.Sprintf("</tr>\n"))
	}
	html.WriteString(fmt.Sprintf("</tbody>\n"))
	html.WriteString(fmt.Sprintf("</table>\n"))

	return html.String()
}

func PrintVersion() {
	fmt.Printf("rabbitmq-timeline version %s (%s)\n", BuildVersion, BuildDate)
}

func PrintUsage() {
	fmt.Printf("Usage: rabbitmq-timeline FILE1 FILE2 FILE3... > FILE\n\n")
}

func main() {

	logPattern, _ := regexp.Compile(`(?P<date>[0-9-]{10}) (?P<time>[0-9.:]{12}) \[(?P<severity>[a-z]+)\] <(?P<pid>[0-9.]+)> (?P<message>.*)`)

	args := os.Args
	inputFiles := args[1:]

	if len(inputFiles) == 0 {
		PrintVersion()
		PrintUsage()
		os.Exit(0)
	}

	nodes := NewNodes(inputFiles)

	var logMessages []LogMessage
	var logDateTimes []string

	for nodeIndex, node := range nodes {
		f, _ := os.Open(node.FileName)
		fs := bufio.NewScanner(f)
		for fs.Scan() {
			logLine := fs.Text()
			matches := logPattern.FindStringSubmatch(logLine)

			// If there were no matches then this must be part of multiline log
			// So append to previous log message
			if len(matches) == 0 {
				logMessages[len(logMessages)-1].Message = append(logMessages[len(logMessages)-1].Message, logLine)
				continue
			}

			newLogMessage := NewLogMessageFromLine(matches)
			newLogMessage.Node = nodeIndex
			logMessages = append(logMessages, newLogMessage)

			// Track every timestamp to help with printing HTML table later
			logDateTimes = append(logDateTimes, newLogMessage.DateTime)
		}
	}

	for index, _ := range logMessages {
		checkLogMessageForReport(&logMessages[index], nodes)
	}

	logDateTimes = RemoveDuplicatesFromSlice(logDateTimes)
	sort.Strings(logDateTimes)

	var logTable = make(map[string][][]*LogMessage)
	for _, dateTime := range logDateTimes {
		logTable[dateTime] = make([][]*LogMessage, len(nodes))
	}

	for index, logMessage := range logMessages {
		logTable[logMessage.DateTime][logMessage.Node] = append(logTable[logMessage.DateTime][logMessage.Node], &logMessages[index])
	}

	html := generateReportHTML(logTable, logDateTimes, nodes)

	fmt.Println(html)
}
