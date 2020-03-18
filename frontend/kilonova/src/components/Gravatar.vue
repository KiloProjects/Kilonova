<template>
    <img :src="url" :alt="alt" />
</template>
<script type="ts">
import Vue from 'vue';
import md5 from "md5";
export default Vue.extend({
    name: "gravatar",
    props: {
        email: {
            type: String,
            default: "",
        },
        size: {
            type: Number,
            default: 80,
        },
        defaultImg: {
            type: String,
            default: "retro",
        },
        alt: {
            type: String,
            default: "Gravatar Picture",
        },
    },
    computed: {
        url: function() {
            console.log(this.email);
            const img = `https://www.gravatar.com/avatar/${this.hash ||
                md5(this.email.trim().toLowerCase())}?s=${this.size}&d=${
                this.defaultImg
            }`;

            return img;
        },
    },
});
</script>
