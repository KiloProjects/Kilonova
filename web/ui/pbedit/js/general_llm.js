                async function translateStatement(e) {
                    e.preventDefault()
                    if(!(await bundled.confirm(bundled.getText("translateStatementWarning")))) {
                        return
                    }
                    let res = await bundled.postCall(`/problem/${problem.id}/update/translateStatement`, {})

                    bundled.apiToast(res)
                }
                document.getElementById("translateStatementForm").addEventListener("submit", translateStatement)
