<template>
    <div>
        <h1>USERS:</h1>
        <p v-for="user in users" :key="'user' + user.ID">
            <router-link :to="'/profile/' + user.name">{{
                user.name
            }}</router-link>
            {{ user }}
        </p>

        <h1>MOTDS:</h1>
        <p v-for="motd in motds" :key="'motd' + motd.ID">{{ motd.Motd }}</p>

        <h1>PROBLEMS:</h1>
        <p v-for="problem in problems" :key="'problem' + problem.ID">
            <router-link :to="'/problem/' + problem.ID">{{
                problem.name
            }}</router-link>
            {{ problem }}
        </p>
    </div>
</template>
<script>
import Vue from "vue";
import axios from "axios";
import defines from "../defines";
export default Vue.extend({
    name: "main-page",
    data: function() {
        return {
            users: [],
            motds: [],
            problems: [],
        };
    },
    created: async function() {
        try {
            let data = await axios.get(defines.prefixURL + "admin/getUsers");
            console.log(data.data);
            this.users = data.data;
        } catch (e) {
            console.error(e);
        }
        try {
            let data = await axios.get(defines.prefixURL + "motd/getAll");
            console.log(data.data);
            this.motds = data.data;
        } catch (e) {
            console.error(e);
        }
        try {
            let data = await axios.get(defines.prefixURL + "problem/getAll");
            console.log(data.data);
            this.problems = data.data;
        } catch (e) {
            console.error(e);
        }
    },
});
</script>
