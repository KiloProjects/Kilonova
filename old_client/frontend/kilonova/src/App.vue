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
                    <b-navbar-brand to="/logout">
                        Log Out
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
import Axios from 'axios';
import defines from './defines';

export default Vue.extend({
    name: "app",
    created: async function() {
        let data = await Axios.get(defines.prefixURL + "user/getSelf")
        console.log(data)
    }
});
</script>

<style src="normalize.css/normalize.css"></style>
