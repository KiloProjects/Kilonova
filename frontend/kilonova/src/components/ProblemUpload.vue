<template>
    <b-form @submit="onSubmit">
        <editor
            :code="code"
            :options="codemirrorOpts"
            @input="onInput"
        ></editor>
        <b-button type="submit">Submit</b-button>
    </b-form>
</template>
<script>
import Vue from "vue";
import { codemirror } from "vue-codemirror";
import defines from "../defines";

Vue.component("editor", codemirror);
export default {
    name: "problem-upload",
    props: {
        id: {
            default: -1,
        },
        language: {
            type: String,
            default: "cpp",
        },
    },
    data: function() {
        return {
            code: "",
            codemirrorOpts: {
                tabsize: 4,
                mode: defines.CMlangs[this.language],
                theme: "monokai",
                lineNumbers: true,
                line: true,
                autoCloseBrackets: true,
                keyMap: "sublime",
                styleActiveLine: true,
            },
        };
    },
    methods: {
        onInput: function(newCode) {
            this.code = newCode;
        },
        onSubmit: function(e) {
            e.preventDefault();
            console.log(e);
        },
    },
};
</script>
