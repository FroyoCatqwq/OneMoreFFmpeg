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
