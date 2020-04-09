<template>
    <b-form @submit.prevent="userLogin">
        <b-input
            v-model="login.username"
            type="text"
            placeholder="username"
            required
        />
        <b-input
            v-model="login.password"
            type="password"
            placeholder="password"
            required
        />
        <b-button type="submit">Log In</b-button>
        <p>{{ response }}</p>
    </b-form>
</template>
<script>
export default {
    data() {
        return {
            login: {
                username: '',
                password: ''
            },
            response: ''
        }
    },
    methods: {
        async userLogin() {
            try {
                await this.$auth.loginWith('local', {
                    params: this.login
                })
            } catch (err) {
                this.response = err.response.data
            }
        }
    }
}
</script>
