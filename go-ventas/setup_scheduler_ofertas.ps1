
$taskName = "VentasMensualesOFERTAS"
$exePath = "C:\Users\Alkosto\Desktop\excel - automatico\go-ventas\ventas_mensuales.exe"
$workDir = "C:\Users\Alkosto\Desktop\excel - automatico\go-ventas"
$action = New-ScheduledTaskAction -Execute $exePath -Argument '--auto-ofertas --mode=IP' -WorkingDirectory $workDir
$triggers = @(New-ScheduledTaskTrigger -Once -At '00:00' -RepetitionInterval (New-TimeSpan -Minutes 5) -RepetitionDuration (New-TimeSpan -Days 3650))
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable -MultipleInstances IgnoreNew
$loggedUser = (Get-CimInstance -ClassName Win32_ComputerSystem).UserName
if (-not $loggedUser) { $loggedUser = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name }
$principal = New-ScheduledTaskPrincipal -UserId $loggedUser -LogonType Interactive -RunLevel Highest

try {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Register-ScheduledTask -TaskName $taskName -Principal $principal -Trigger $triggers -Action $action -Settings $settings
    Write-Host "Tarea OFERTAS creada: $taskName (cada 5 minutos, 24/7)"
} catch {
    Write-Host "ERROR: $_"
}
