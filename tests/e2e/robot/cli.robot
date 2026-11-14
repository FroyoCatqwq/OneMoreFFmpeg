*** Settings ***
Library    Process
Library    OperatingSystem
Suite Setup    Set Environment Variable    ONE_MORE_FFMPEG_HOME    ${TEMPDIR}${/}omf-robot

*** Variables ***
${CLI}    %{OMF_BINARY}

*** Test Cases ***
Version Returns JSON
    ${result}=    Run Process    ${CLI}    version    --json
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain    ${result.stdout}    "onemoreffmpeg.version/v1"

Provider List Is Offline
    ${result}=    Run Process    ${CLI}    provider    list    --json
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain    ${result.stdout}    "btbn"

Doctor Returns Report
    ${result}=    Run Process    ${CLI}    doctor    --json
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain    ${result.stdout}    "onemoreffmpeg.doctor/v1"

Doctor Strict Unhealthy Exits Nonzero
    ${result}=    Run Process    ${CLI}    doctor    --strict    --json
    Should Be Equal As Integers    ${result.rc}    1
    Should Contain    ${result.stdout}    OMF-E6006

Env System Consent Dry Run
    ${result}=    Run Process    ${CLI}    env    set    --system    --authorize-system    --json
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain    ${result.stdout}    "dry_run":true
