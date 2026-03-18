@echo off
setlocal enabledelayedexpansion

set GOPRIVATE=github.com/microfaults/*

set SERVICES=src\emailservice src\productcatalogservice src\recommendationservice src\shoppingassistantservice src\shippingservice src\checkoutService src\paymentservice src\currencyservice src\cartservice\src src\frontend src\adservice

set SCRIPT_DIR=%~dp0

for %%S in (%SERVICES%) do (
    if exist "%SCRIPT_DIR%%%S\go.mod" (
        echo ==^> Vendoring %%S
        pushd "%SCRIPT_DIR%%%S"
        go mod vendor
        if !errorlevel! neq 0 (
            echo ==^> FAILED to vendor %%S
            popd
            exit /b 1
        )
        popd
    ) else (
        echo ==^> Skipping %%S ^(no go.mod found^)
    )
)

echo ==^> All services vendored successfully
endlocal
