package services

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/mic-360/wimo/internal/state"
)

type OptimizerService struct {
	logger *Logger
}

func NewOptimizerService(logger *Logger) *OptimizerService {
	return &OptimizerService{logger: logger}
}

func (o *OptimizerService) Tasks() []state.OptimizeTask {
	return []state.OptimizeTask{
		{ID: "flush-dns", Category: "Network", Title: "Flush DNS cache", Description: "Clear resolver cache and refresh local lookups", Selected: true, Command: "Clear-DnsClientCache"},
		{ID: "register-dns", Category: "Network", Title: "Re-register DNS", Description: "Refresh local DNS registration", Selected: false, Command: "ipconfig /registerdns | Out-Null"},
		{ID: "clear-recycle", Category: "Storage", Title: "Empty Recycle Bin", Description: "Permanently clear items already marked for deletion", Selected: false, Command: "Clear-RecycleBin -Force -ErrorAction SilentlyContinue"},
		{ID: "clear-thumbs", Category: "Storage", Title: "Rebuild thumbnail cache", Description: "Drop Explorer thumbnail databases so they can regenerate", Selected: false, Command: "Remove-Item \"$env:LOCALAPPDATA\\Microsoft\\Windows\\Explorer\\thumbcache_*.db\" -Force -ErrorAction SilentlyContinue"},
		{ID: "trim-ssd", Category: "Storage", Title: "Trim SSD volumes", Description: "Run retrim on fixed disks that support it", Selected: false, Command: "Get-Volume | Where-Object { $_.DriveType -eq 'Fixed' -and $_.DriveLetter } | ForEach-Object { Optimize-Volume -DriveLetter $_.DriveLetter -ReTrim -ErrorAction SilentlyContinue }"},
		{ID: "search-service", Category: "System", Title: "Restart Windows Search", Description: "Refresh indexing service state", Selected: false, Command: "Restart-Service WSearch -ErrorAction SilentlyContinue"},
		{ID: "shader-cache", Category: "Graphics", Title: "Clear shader caches", Description: "Drop NVIDIA, AMD and Direct3D caches", Selected: false, Command: "$paths=@('$env:LOCALAPPDATA\\NVIDIA\\DXCache','$env:LOCALAPPDATA\\NVIDIA\\GLCache','$env:LOCALAPPDATA\\AMD\\DxCache','$env:LOCALAPPDATA\\D3DSCache'); foreach($p in $paths){ if(Test-Path $p){ Remove-Item \"$p\\*\" -Recurse -Force -ErrorAction SilentlyContinue }}"},
		{ID: "delivery-optimization", Category: "System", Title: "Clear Delivery Optimization cache", Description: "Remove cached update delivery data", Selected: false, AdminOnly: true, Command: "Delete-DeliveryOptimizationCache -Force -ErrorAction SilentlyContinue"},
		{ID: "winsock", Category: "Network", Title: "Reset Winsock catalog", Description: "Repair network stack registrations", Selected: false, AdminOnly: true, Command: "netsh winsock reset | Out-Null"},
		{ID: "visual-effects", Category: "Interface", Title: "Reduce visual effects", Description: "Switch Windows visual effects to the best performance preset", Selected: false, Command: "$path='HKCU:\\SOFTWARE\\Microsoft\\Windows\\CurrentVersion\\Explorer\\VisualEffects'; Set-ItemProperty -Path $path -Name VisualFXSetting -Value 2 -ErrorAction SilentlyContinue"},
	}
}

func (o *OptimizerService) Run(ctx context.Context, tasks []state.OptimizeTask) (OperationReport, []state.OptimizeTask, error) {
	updated := make([]state.OptimizeTask, len(tasks))
	copy(updated, tasks)
	report := OperationReport{Title: "Optimize", Message: "Optimization complete"}
	admin := probeAdmin()
	for index := range updated {
		if !updated[index].Selected {
			continue
		}
		if updated[index].AdminOnly && !admin {
			updated[index].Status = "skipped"
			updated[index].LastResult = "administrator rights required"
			report.Errors = append(report.Errors, updated[index].Title+": administrator rights required")
			continue
		}
		started := time.Now()
		err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command", updated[index].Command).Run()
		updated[index].Duration = time.Since(started)
		if err != nil {
			updated[index].Status = "failed"
			updated[index].LastResult = err.Error()
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", updated[index].Title, err))
			continue
		}
		updated[index].Status = "done"
		updated[index].LastResult = fmt.Sprintf("completed in %s", updated[index].Duration.Round(10*time.Millisecond))
		report.Count++
	}
	return report, updated, nil
}
