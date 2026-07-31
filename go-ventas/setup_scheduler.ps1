
$taskName = "VentasMensuales"
$action = New-ScheduledTaskAction -Execute "C:\Users\Administrador\Desktop\Reportes FA\go-ventas\ventas_mensuales.exe" -Argument '--auto --mode=IP'
$trigger = New-ScheduledTaskTrigger -Once -At "09:00" -RepetitionInterval (New-TimeSpan -Minutes 120) -RepetitionDuration (New-TimeSpan -Hours 8)
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
$loggedUser = (Get-CimInstance -ClassName Win32_ComputerSystem).UserName
$principal = New-ScheduledTaskPrincipal -UserId $loggedUser -LogonType Interactive -RunLevel Highest

try {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Register-ScheduledTask -TaskName $taskName -Principal $principal -Trigger $trigger -Action $action -Settings $settings
    Write-Host "Tarea creada: $taskName (09:00 a 17:00, cada 120 min)"
} catch {
    Write-Host "ERROR: $_"
}
