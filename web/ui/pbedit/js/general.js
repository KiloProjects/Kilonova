            let problem = __PROBLEM__;
            async function updateVisibility(e) {
                e.preventDefault();
                const visible = document.getElementById("visibility_check").checked
                let res = await bundled.postCall(`/problem/${problem.id}/update/`, { visible })
                bundled.apiToast(res)
            }
            document.getElementById("visibility_form").addEventListener("submit", updateVisibility)

            async function requestReview(e) {
                e.preventDefault()
                if(!(await bundled.confirm("Are you sure you want to request a review? Make sure the problem is fully ready first!"))) {
                    return;
                }

                let result = await bundled.postCall(`/problem/${problem.id}/update/`, {
                    review_requested: true,
                });
                if(result.status === "success") {
                    window.location.reload();
                    return
                }
                bundled.apiToast(result);
            }
            document.getElementById("review_request_btn").addEventListener("click", requestReview)

