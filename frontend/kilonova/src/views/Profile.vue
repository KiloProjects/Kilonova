<template>
    <div>
        <br />
        <gravatar :email="user.email" size="160" />
        <h1>
            <b-badge>#{{ user.ID }}</b-badge>
            <span> {{ user.name }}</span>
        </h1>
        <h4>Misc data:</h4>
        <h6></h6>
        {{ user }}
    </div>
</template>

<script type="ts">
import Vue from 'vue';
import axios from "axios";
import defines from "../defines";
import "../components/Gravatar";
export default Vue.extend({
    name: "profile-page",
    data: function() {
        return {
            user: {},
        };
    },
    created: async function() {
        let req = await axios.get(
            defines.prefixURL + "getUserByName/" + this.$route.params.name
        );
        this.$data.user = req.data;
    },
});
</script>
