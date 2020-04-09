<template>
    <div>
        <ul v-if="probleme">
            <li v-for="problema in probleme" :key="problema.ID">
                {{ problema }}
            </li>
        </ul>
        <ErrorCard :err="error" />
    </div>
</template>
<script>
import ErrorCard from '~/components/ErrorCard'
export default {
    components: {
        ErrorCard
    },
    data() {
        return {
            probleme: null,
            error: null
        }
    },
    async created() {
        try {
            const probleme = await this.$axios.get('/problem/getAll')
            this.probleme = probleme.data
        } catch (err) {
            this.error = err.response.data
        }
    }
}
</script>
