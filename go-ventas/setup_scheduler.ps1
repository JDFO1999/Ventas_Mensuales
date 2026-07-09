
$taskName = "VentasMensuales"
$action = New-ScheduledTaskAction -Execute "cmd.exe" -Argument ('/c "' + "C:\Users\Alkosto\Desktop\excel - automatico\go-ventas\run_ventas.cmd" + '"')
$trigger = New-ScheduledTaskTrigger -Once -At "08:00" -RepetitionInterval (New-TimeSpan -Minutes 60) -RepetitionDuration (New-TimeSpan -Hours 10)
$settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable

try {
    Unregister-ScheduledTask -TaskName $taskName -Confirm:$false -ErrorAction SilentlyContinue
    Register-ScheduledTask -TaskName $taskName -Trigger $trigger -Action $action -Settings $settings -RunLevel Highest
    Write-Host "Tarea creada: $taskName (08:00 a 18:00, cada 60 min)"
} catch {
    Write-Host "ERROR: $_"
}
