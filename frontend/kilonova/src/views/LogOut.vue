<template>
    <div>
        {{ output }}
    </div>
</template>
<script>
import Axios from "axios";
export default {
    name: "logout",
    data: function() {
        return {
            output: "",
        };
    },
    created: async function() {
        try {
            await Axios.get("/api/auth/logout");
            this.$router.push("/");
        } catch (e) {
            this.output = e.response;
        }
    },
};
</script>
