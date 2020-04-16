<template>
    <b-form @submit.prevent="userSignup">
        <b-input v-model="signup.username" type="text" placeholder="username" />
        <b-input v-model="signup.email" type="email" placeholder="email" />
        <b-input
            v-model="signup.password"
            type="password"
            placeholder="password"
            autocomplete="new-password"
        />
        <b-button type="submit">Sign Up</b-button>
        <p>{{ response }}</p>
    </b-form>
</template>
<script>
export default {
    data() {
        return {
            signup: {
                username: '',
                email: '',
                password: ''
            },
            response: ''
        }
    },
    methods: {
        async userSignup() {
            try {
                await this.$axios({
                    method: 'POST',
                    url: '/auth/signup',
                    params: this.signup
                })
                await this.$auth.loginWith('local', { params: this.signup })
                this.$router.push('/')
            } catch (err) {
                this.response = err.response.data
            }
        }
    },
    middleware: 'auth',
    auth: 'guest'
}
</script>
