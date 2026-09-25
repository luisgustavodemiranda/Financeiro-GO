# Carrega as credenciais locais protegidas por DPAPI, sem exibir senhas.
# Execute no mesmo terminal em que rodará go run ou go test.
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$DatabaseHost,
    [ValidateSet('Application', 'Test')][string]$Target = 'Application',
    [ValidateRange(1, 65535)][int]$Port = 5432,
    [ValidateSet('disable', 'require', 'verify-ca', 'verify-full')][string]$SSLMode = 'require'
)

$ErrorActionPreference = 'Stop'
$credentialPath = Join-Path (Split-Path -Parent $PSScriptRoot) '.env.postgres.clixml'
$saved = Import-Clixml -LiteralPath $credentialPath
$credential = if ($Target -eq 'Test') { $saved.Test } else { $saved.App }
if ($credential -isnot [PSCredential]) { throw 'Credencial local inválida ou indisponível para este usuário Windows.' }
$databaseName = if ($Target -eq 'Test') { 'financeiro_go_test' } else { 'financeiro_go' }
$expectedUser = if ($Target -eq 'Test') { 'financeiro_go_test_app' } else { 'financeiro_go_app' }
if ($credential.UserName -ne $expectedUser) { throw 'Usuário local não corresponde ao destino escolhido.' }

Remove-Item Env:FINANCEIRO_DATABASE_URL -ErrorAction SilentlyContinue
$env:DB_HOST = $DatabaseHost
$env:DB_PORT = [string]$Port
$env:DB_NAME = $databaseName
$env:DB_USER = $credential.UserName
$env:DB_PASSWORD = $credential.GetNetworkCredential().Password
$env:DB_SSLMODE = $SSLMode
$env:PERSISTENCE = 'postgres'
if ($Target -eq 'Test') {
    $builder = [UriBuilder]::new('postgresql', $DatabaseHost, $Port, $databaseName)
    $builder.UserName = [Uri]::EscapeDataString($env:DB_USER)
    $builder.Password = [Uri]::EscapeDataString($env:DB_PASSWORD)
    $builder.Query = 'sslmode=' + $SSLMode
    $env:FINANCEIRO_TEST_DATABASE_URL = $builder.Uri.AbsoluteUri
} else {
    Remove-Item Env:FINANCEIRO_TEST_DATABASE_URL -ErrorAction SilentlyContinue
}
Write-Host "Ambiente PostgreSQL preparado para $databaseName."
