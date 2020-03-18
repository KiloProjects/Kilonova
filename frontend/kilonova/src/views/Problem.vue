<template>
    <div v-if="problemData.name != ''">
        <h1>
            <b-badge>#{{ problemData.ID }}</b-badge>
            <span> {{ problemData.name }}</span>
        </h1>
        <h4>Misc:</h4>
        <h6>test name: {{ problemData.testName }}</h6>
        <h6>created at: {{ problemData.CreatedAt }}</h6>
        <h6>updated at: {{ problemData.UpdatedAt }}</h6>
        <h6>deleted at: {{ problemData.DeletedAt }}</h6>
        <h6></h6>
        <h3>Description:</h3>
        <p>
            {{ problemData.text }}
        </p>
        <br />
        <h4>Tests:</h4>
        <p v-if="problemData.tests == null || !problemData.tests.length">
            Tests not found
        </p>
        <div v-if="problemData.tests != null && problemData.tests.length">
            <p v-for="test in problemData.tests" :key="test.ID">
                {{ test }}
            </p>
        </div>

        <problemupload :id="$route.params.id"></problemupload>
    </div>
</template>
<script>
import Vue from "vue";
import axios from "axios";
import defines from "../defines";
export default Vue.extend({
    name: "problem",
    data: function() {
        return {
            problemData: {},
        };
    },
    created: async function() {
        let url = defines.prefixURL + "getProblemByID/" + this.$route.params.id;
        let data = await axios.get(url);
        this.$data.problemData = data.data;
    },
});
</script>
