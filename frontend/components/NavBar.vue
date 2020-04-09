<template>
    <b-navbar toggleable="lg" type="dark" variant="primary">
        <b-navbar-brand href="/">KiloNova</b-navbar-brand>
        <b-navbar-toggle target="nav-collapse"></b-navbar-toggle>
        <b-collapse id="nav-collapse" is-nav>
            <b-navbar-nav class="mr-auto">
                <b-nav-item-dropdown text="Actiuni">
                    <b-dropdown-item href="/probleme">Probleme</b-dropdown-item>
                    <b-dropdown-item href="/profile">Profil</b-dropdown-item>
                </b-nav-item-dropdown>
            </b-navbar-nav>
            <b-navbar-nav class="ml-auto">
                <template v-if="!isAuthed">
                    <b-nav-item href="/signup">Sign Up</b-nav-item>
                    <b-nav-item href="/login">Log In</b-nav-item>
                </template>
                <template v-else>
                    <b-nav-item @click="logout">Log Out</b-nav-item>
                </template>
            </b-navbar-nav>
        </b-collapse>
    </b-navbar>
</template>
<script>
import { mapGetters } from 'vuex'
export default {
    computed: {
        ...mapGetters(['isAuthed', 'loggedInUser'])
    },
    methods: {
        async logout() {
            await this.$auth.logout()
        }
    }
}
</script>
