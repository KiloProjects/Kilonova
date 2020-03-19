<template>
    <div>
        <b-form @submit="onSubmit">
            <b-input type="text" v-model="username" placeholder="username" />
            <b-input type="password" v-model="password" placeholder="parola" />
            <b-button type="submit">Logare</b-button>
        </b-form>
        <p>{{ output }}</p>
    </div>
</template>
<script>
import Vue from "vue";
import Axios from "axios";
export default Vue.extend({
    name: "login",
    data: function() {
        return {
            username: "",
            password: "",
        };
    },
    methods: {
        onSubmit: async function(e) {
            e.preventDefault();
            try {
                await Axios({
                    url: "/api/auth/login",
                    method: "POST",
                    params: {
                        username: this.username,
                        password: this.password,
                    },
                });
                this.$router.push("/");
            } catch (err) {
                this.output = err.response.data;
                console.log(err);
            }
        },
    },
});
</script>
