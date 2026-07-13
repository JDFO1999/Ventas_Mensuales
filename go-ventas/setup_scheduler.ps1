$taskName = "VentasMensuales"
$action = New-ScheduledTaskAction -Execute "cmd.exe" -Argument ('/c "C:\Users\Alkosto\Desktop\excel - automatico\go-ventas\run_ventas.cmd"')
$trigger = New-ScheduledTaskTrigger -Daily -At "06:00"
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
$loggedUser = (Get-CimInstance -ClassName Win32_ComputerSystem).UserName
$principal = New-ScheduledTaskPrincipal -UserId $loggedUser -LogonType Interactive -RunLevel Highest

try {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Register-ScheduledTask -TaskName $taskName -Principal $principal -Trigger $trigger -Action $action -Settings $settings
    Write-Host "Tarea creada: $taskName (diaria 06:00) para $loggedUser"
} catch {
    Write-Host "ERROR: $_"
}
