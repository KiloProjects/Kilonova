<template>
    <div>{{ userString }}</div>
</template>
<script>
export default {
    props: {
        user: {
            type: Object,
            default: () => {}
        }
    },
    computed: {
        userString() {
            return JSON.stringify(this.$props.user, null, 4)
        }
    }
}
</script>
