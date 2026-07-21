
$taskName = "VentasMensualesCA"
$action = New-ScheduledTaskAction -Execute "C:\Users\Alkosto\Desktop\excel - automatico\go-ventas\ventas_mensuales.exe" -Argument "--auto-ca --mode=IP"
$trigger = New-ScheduledTaskTrigger -Once -At "08:00" -RepetitionInterval (New-TimeSpan -Minutes 1440) -RepetitionDuration (New-TimeSpan -Hours 4)
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
$loggedUser = (Get-CimInstance -ClassName Win32_ComputerSystem).UserName
$principal = New-ScheduledTaskPrincipal -UserId $loggedUser -LogonType Interactive -RunLevel Highest

try {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Register-ScheduledTask -TaskName $taskName -Principal $principal -Trigger $trigger -Action $action -Settings $settings
    Write-Host "Tarea CA creada: $taskName (08:00 a 12:00)"
} catch {
    Write-Host "ERROR: $_"
}
