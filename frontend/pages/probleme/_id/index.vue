<template>
    <div>
        <Problem v-if="data" :problem="data" />
        <b-card v-if="error" bg-variant="danger"> </b-card>
    </div>
</template>
<script>
import Problem from '~/components/Problem'
export default {
    validate({ params }) {
        return /^\d+$/.test(params.id)
    },
    components: {
        Problem
    },
    async asyncData({ params, $axios }) {
        try {
            const data = await $axios.get('/problem/getByID', {
                params: { id: params.id }
            })
            return {
                data: data.data.data
            }
        } catch (err) {
            return {
                error: err.response.data
            }
        }
    },
    data() {
        return {
            data: null,
            error: null
        }
    }
}
</script>
