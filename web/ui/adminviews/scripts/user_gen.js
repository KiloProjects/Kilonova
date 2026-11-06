        function strOrUndefinedValue(id) {
            let val = document.getElementById(id).value;
            if (typeof val !== "string" || val.length == 0) {
                return undefined;
            }
            return val;
        }

        async function generateUser(e) {
            e.preventDefault()

            let contestID = parseInt(document.getElementById("contestID").value);
            if (isNaN(contestID)) {
                contestID = undefined;
            }

            let data = {
                username: document.getElementById("username").value,
                password: strOrUndefinedValue("password"),
                language: strOrUndefinedValue("langSelect"),

                bio: strOrUndefinedValue("bio"),

                email: strOrUndefinedValue("email"),
                display_name: strOrUndefinedValue("display_name"),
                contest_id: contestID,

                password_by_mail: document.getElementById("pwd_mail_enable").checked,
                password_by_mail_to: strOrUndefinedValue("pwd_email_address"),

                mail_subject: strOrUndefinedValue("pwd_email_subject"),
            }

            let res = await bundled.postCallV2("/admin/v2/generateUser", data)
            if (res.status === "error") {
                bundled.apiToast(res)
                return
            }

            let output = document.createElement("div")
            output.classList.add("reset-list", "segment-panel", "my-2")
            output.innerHTML = `<ul>
                <li>User: <a href="/profile/${res.data.user.name}">${res.data.user.name}</a></li>
                <li>${bundled.getText("password")}: <code>${res.data.password}</code></li>
            </ul>`

            document.getElementById("output").appendChild(output)

        }
        document.getElementById("userGenForm").addEventListener("submit", generateUser)
