<template>
    <div>
        <b-form @submit="onSubmit">
            <b-input
                type="username"
                v-model="username"
                placeholder="Nume..."
            ></b-input>
            <b-input
                type="email"
                v-model="email"
                placeholder="Email..."
            ></b-input>
            <b-input
                type="password"
                v-model="password"
                placeholder="Parola..."
                :state="validation"
            ></b-input>
            <b-input
                type="password"
                v-model="password1"
                placeholder="Parola din nou..."
                :state="validation"
            ></b-input>
            <b-form-invalid-feedback :state="validation">
                Passwords not ok
            </b-form-invalid-feedback>
            <b-form-valid-feedback :state="validation">
                Passwords ok
            </b-form-valid-feedback>
            <b-button type="submit">Inregistrare</b-button>
        </b-form>
        {{ output }}
    </div>
</template>

<script>
import defines from "../defines";
import Axios from "axios";
// import * as $ from "jquery";

export default {
    name: "signup",
    data: function() {
        return {
            username: "",
            email: "",
            password: "",
            password1: "",
            output: "",
        };
    },
    methods: {
        onSubmit: async function(e) {
            e.preventDefault();
            if (this.password != this.password1) {
                this.output = "Username and password dont match";
            }
            try {
                let data = await Axios({
                    url: defines.prefixURL + "auth/signup",
                    method: "POST",
                    params: {
                        username: this.username,
                        email: this.email,
                        password: this.password,
                    },
                });
                console.log(data);
                this.output = data.data;
                this.$router.push("/");
            } catch (err) {
                this.output = `Error status code ${err.response.status} (${err.response.statusText}): ${err.response.data}`;
                console.log(err);
            }
        },
    },
    computed: {
        validation: function() {
            return (
                this.password1 == "" ||
                (this.password == this.password1 && this.password.length > 5)
            );
        },
    },
};
</script>
