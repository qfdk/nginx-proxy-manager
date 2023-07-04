package controllers

import (
	"github.com/gin-gonic/gin"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

var TtydProcess *os.Process

//	func Terminal(ctx *gin.Context) {
//		ctx.HTML(http.StatusOK, "terminal.html", gin.H{})
//	}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func waitForServerReady(url string) {
	for {
		resp, err := http.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
}
func isCommandAvailable(name string) bool {
	cmd := exec.Command("which", name)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func installTtyd() error {
	cmd := exec.Command("sudo", "apt-get", "update")
	err := cmd.Run()
	if err != nil {
		return err
	}

	cmd = exec.Command("sudo", "apt-get", "install", "-y", "ttyd")
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}

func TerminalStart(ctx *gin.Context) {
	if err := ensureTtydInstalled(); err != nil {
		ctx.String(http.StatusInternalServerError, "Error preparing ttyd: %v", err)
		return
	}

	if err := addIptablesRule(); err != nil {
		ctx.String(http.StatusInternalServerError, "Error adding iptables rule: %v", err)
		return
	}

	shell := findAvailableShell()

	// 创建一个命令对象
	cmd := exec.Command("ttyd", "-t", "cursorStyle=bar", "-t", "lineHeight=1.2", "-t", "fontSize=14", shell)

	// 开启新的会话以创建守护进程
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		log.Println("Error starting command:", err)
		ctx.String(http.StatusInternalServerError, "Error starting command: %v", err)
		return
	}

	TtydProcess = cmd.Process
	log.Println("ttyd has been started with PID:", TtydProcess.Pid)
	waitForServerReady("http://" + getLocalIP() + ":7681/")
	ctx.Redirect(http.StatusFound, "http://"+getLocalIP()+":7681/")
}

func ensureTtydInstalled() error {
	if isCommandAvailable("ttyd") {
		return nil
	}

	log.Println("`ttyd` command is not available, trying to install it...")
	err := installTtyd()
	if err != nil {
		log.Println("Failed to install `ttyd`: ", err)
		return err
	}

	log.Println("`ttyd` has been installed successfully.")
	return nil
}

func addIptablesRule() error {
	log.Println("检查 iptables 规则")

	output, err := exec.Command("sh", "-c", "sudo iptables -L | grep 'dpt:7681'").Output()
	if err != nil {
		log.Println("Error reading iptables rules:", err)
		return err
	}

	// 如果已经有相应的规则，不再添加
	if strings.Contains(string(output), "ACCEPT     tcp  --  anywhere             anywhere             tcp dpt:7681") {
		log.Println("iptables rule for port 7681 already exists")
		return nil
	}

	log.Println("执行 iptables 添加规则命令")
	iptablesCmd := exec.Command("sudo", "iptables", "-I", "INPUT", "-p", "tcp", "--dport", "7681", "-j", "ACCEPT")
	err = iptablesCmd.Run()
	if err != nil {
		log.Println("Error adding iptables rule:", err)
		return err
	}

	return nil
}

func findAvailableShell() string {
	if isCommandAvailable("zsh") {
		return "zsh"
	}

	return "bash"
}

func TerminalStop(ctx *gin.Context) {
	if TtydProcess != nil {
		err := TtydProcess.Kill()
		if err != nil {
			log.Println("Error stopping command:", err)
			ctx.String(http.StatusInternalServerError, "Error stopping command: %v", err)
			return
		}
		TtydProcess = nil
		log.Println("ttyd has been stopped")
	} else {
		log.Println("ttyd is not running")
	}
	ctx.Redirect(http.StatusFound, "/")
}
