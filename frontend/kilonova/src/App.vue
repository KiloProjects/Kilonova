<template>
    <div>
        <div>
            <b-navbar toggleable="lg" type="dark" variant="primary">
                <b-navbar-brand to="/">
                    KiloNova
                </b-navbar-brand>
                <b-navbar-nav align="left">
                    <b-navbar-brand to="/signup">
                        Sign Up
                    </b-navbar-brand>
                    <b-navbar-brand to="/login">
                        Log in
                    </b-navbar-brand>
                    <b-navbar-brand to="/createProblem">
                        Create Problem
                    </b-navbar-brand>
                </b-navbar-nav>
            </b-navbar>
        </div>
        <b-container>
            <router-view></router-view>
        </b-container>
    </div>
</template>

<script type="ts">
import Vue from 'vue';

export default Vue.extend({
    name: "app",
});
</script>

<style src="normalize.css/normalize.css"></style>
