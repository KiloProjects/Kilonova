<template>
    <div>
        <Problem v-if="data" :problem="data" />
        <b-card v-if="error" bg-variant="danger"> </b-card>
    </div>
</template>
<script>
export default {
    validate({ params }) {
        return /^\d+$/.test(params.id)
    },
    data() {
        return {
            data: null,
            error: null
        }
    },
    async created() {
        try {
            const data = await this.$axios.get('/problem/getByID', {
                params: { id: this.$route.params.id }
            })
            this.data = data.data.data
        } catch (err) {
            this.error = err.response.data
        }
    }
}
</script>
