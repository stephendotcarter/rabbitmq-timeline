package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
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
	for index, logPath := range inputFiles {
		log.Printf("%d %s\n", index, logPath)
		_, err := os.Stat(logPath)
		if err == nil {
			log.Printf("file %s exists", logPath)
			newNode := NewNode(logPath)
			newNodes = append(newNodes, newNode)
		}
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
	row := ""
	row += "<tr>"
	row += fmt.Sprintf("<td>%d</td>", message.Node)
	row += fmt.Sprintf("<td class=\"nowrap\">%s</td>", message.DateTime)
	row += fmt.Sprintf("<td>%s</td>", message.Severity)
	row += fmt.Sprintf("<td>%s</td>", message.Pid)
	row += fmt.Sprintf("<td><pre>%s</pre></td>", strings.Join(message.Message[:], "\n"))
	row += "</tr>"
	return row
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

func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func checkLogMessageForInfo(logMessage *LogMessage, nodes []Node) {
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
		}
		if strings.Contains(message, "RabbitMQ is asked to stop...") {
			logMessage.Reports = append(logMessage.Reports, Report{
				"Stopped via \"rabbitmqctl stop_app\"",
				"info",
			})
		}
		if strings.Contains(message, "SIGTERM received - shutting down") {
			logMessage.Reports = append(logMessage.Reports, Report{
				"Stopped via \"service rabbitmq-server stop\"",
				"info",
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
				"info",
			})
		}
		if strings.Contains(message, "disk resource limit alarm set on node ") {
			logMessage.Reports = append(logMessage.Reports, Report{
				message,
				"info",
			})
		}
	}
}

func generateReportHTML(logTable map[string][][]*LogMessage, logDateTimes []string, nodes []Node) string {
	html := ""
	html += fmt.Sprintf(`
<style>
*{
	font-family:monospace;
	font-size:12px;
}
pre{
	margin:0px;
}
td{
	vertical-align:top;
}
table {
	border-top: 1px solid #CCC;
	border-left: 1px solid #CCC;
}
td,th {
	border-bottom: 1px solid #CCC;
	border-right: 1px solid #CCC;
	padding: 0px;
	vertical-align:top;
}
td > div {
	padding:3px;
}
.nowrap{
	white-space:nowrap;
}
.prewrap{
	white-space:pre-wrap;
}
.severity_info{
	background-color:
}
.severity_notice{
	background-color:#4DA6FF;
}
.severity_warning{
	background-color:#FFA64D;
}
.severity_error{
	background-color:#FF4D4D;
}
</style>`)
	html += fmt.Sprintf("<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\">")

	html += "<thead>"
	html += "<tr>"
	html += "<th></td>"
	for _, node := range nodes {
		html += fmt.Sprintf(`<td>
<div>
<strong>Filename:</strong><br>
%s<br>
<strong>Name:</strong><br>
%s<br>
<strong>RabbitMQ:</strong><br>
%s<br>
<strong>Erlang:</strong><br>
%s<br>
<strong>Cookie Hash:</strong><br>
%s</div>
</td>`, node.FileName, node.Name, strings.Join(node.VersionRabbitMQ, "<br>"), strings.Join(node.VersionErlang, "<br>"), strings.Join(node.CookieHash, "<br>"))

	}
	html += "</tr>"
	html += "</thead>"
	html += "<tbody>"
	for _, logDateTime := range logDateTimes {

		logDateTimeHash := GetMD5Hash(logDateTime)
		html += fmt.Sprintf("<tr class=\"%s\">", logDateTimeHash)
		html += fmt.Sprintf("<td class=\"nowrap\"><div>%s</div></td>", logDateTime)
		for _, nodeLogs := range logTable[logDateTime] {

			html += fmt.Sprintf("<td>")
			for _, nodeLog := range nodeLogs {
				html += fmt.Sprintf("<div class=\"prewrap severity_%s\"><strong>[%s]</strong> %s</div>", nodeLog.Severity, nodeLog.Severity, strings.Join(nodeLog.Message[:], "\n"))
				for _, nodeReport := range nodeLog.Reports {
					html += fmt.Sprintf("<div class=\"prewrap severity_%s\"><strong>REPORT [%s]</strong> %s</div>", nodeReport.Severity, nodeReport.Severity, nodeReport.Message)
				}
			}
			html += fmt.Sprintf("</td>")
		}
		html += "</tr>"
	}
	html += "</tbody>"
	html += fmt.Sprintf("</table>")

	return html
}

func main() {

	logPattern, _ := regexp.Compile(`(?P<date>[0-9-]{10}) (?P<time>[0-9.:]{12}) \[(?P<severity>[a-z]+)\] <(?P<pid>[0-9.]+)> (?P<message>.*)`)

	args := os.Args
	inputFiles := args[1:]

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
		checkLogMessageForInfo(&logMessages[index], nodes)
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
