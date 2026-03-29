package core

import (
	"os"
	"os/exec"
	"path/filepath"
)

// StartAriaDaemon spawns aria2c quietly.
func StartAriaDaemon() error {
	home, _ := os.UserHomeDir()
	dlDir := filepath.Join(home, "Downloads")
	_ = os.MkdirAll(dlDir, os.ModePerm)

	daemonCmd := exec.Command("aria2c", 
		"--enable-rpc", 
		"--rpc-listen-all=false", 
		"--rpc-listen-port=6800",
		"--dir="+dlDir,
		"--max-connection-per-server=16",
		"--split=16",
		"--min-split-size=1M",
		"--continue=true",
		"--daemon=true",
	)
	
	return daemonCmd.Run() // Returns immediately because of --daemon=true
}
