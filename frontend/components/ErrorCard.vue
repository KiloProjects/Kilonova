<template>
    <b-card
        v-if="err"
        bg-variant="danger"
        text-variant="white"
        header="Error"
        class="text-center"
        style="max-width: 20rem; margin: auto;"
    >
        <b-card-body v-text="err"></b-card-body>
    </b-card>
</template>
<script>
export default {
    props: {
        err: {
            type: String,
            default: ''
        }
    }
}
</script>
